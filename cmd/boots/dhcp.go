package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
)

var listenAddr = conf.BOOTPBind

func init() {
	flag.StringVar(&listenAddr, "dhcp-addr", listenAddr, "IP and port to listen on for DHCP.")
}

// ServeDHCP is a useless comment
func ServeDHCP() {
	poolSize := env.Int("BOOTS_DHCP_WORKERS", runtime.GOMAXPROCS(0)/2)
	handler := dhcpHandler{pool: workerpool.New(poolSize)}
	defer handler.pool.Stop()

	err := retry.Do(
		func() error {
			return errors.Wrap(dhcp4.ListenAndServe(listenAddr, handler), "serving dhcp")
		},
	)
	if err != nil {
		mainlog.Fatal(errors.Wrap(err, "retry dhcp serve"))
	}
}

type dhcpHandler struct {
	pool *workerpool.WorkerPool
}

func (d dhcpHandler) ServeDHCP(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	d.pool.Submit(func() { d.serveDHCP(w, req) })
}

func (d dhcpHandler) serveDHCP(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	mac := req.GetCHAddr()
	if conf.ShouldIgnoreOUI(mac.String()) {
		mainlog.With("mac", mac).Info("mac is in ignore list")
		return
	}

	gi := req.GetGIAddr()
	if conf.ShouldIgnoreGI(gi.String()) {
		mainlog.With("giaddr", gi).Info("giaddr is in ignore list")
		return
	}

	metrics.DHCPTotal.WithLabelValues("recv", req.GetMessageType().String(), gi.String()).Inc()
	labels := prometheus.Labels{"from": "dhcp", "op": req.GetMessageType().String()}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))

	circuitID, err := getCircuitID(req)
	if err != nil {
		mainlog.With("mac", mac, "err", err).Info("error parsing option82")
	} else {
		mainlog.With("mac", mac, "circuitID", circuitID).Info("parsed option82/circuitid")
	}

	var j = job.Job{}
	j, err = job.CreateFromDHCP(mac, gi, circuitID)
	if err != nil {
		// Cacher did not find any HW
		mainlog.With("mac", mac, "err", err).Info("retrieved hw is empty. MAC address is unknown to tink")
		metrics.JobsInProgress.With(labels).Dec()
		timer.ObserveDuration()

		// Check if we want to use default workflows
		if os.Getenv("ENABLE_DEFAULT_WORKFLOWS") != "1" {
			// We don't want default workflows, so just return
			return
		} else {
			mainlog.With("mac", mac).Info("Default workflow enabled")
			// We want default workflows, so at first we will need to create a hardware
			mainlog.With("mac", mac).Info("Pushing new HW to Tink")
			j, err = job.CreateHWFromDHCP(mac, gi, circuitID)
			if err != nil {
				mainlog.With("mac", mac).Error(err, "failed to create hw")
				metrics.JobsInProgress.With(labels).Dec()
				timer.ObserveDuration()
				return
			}

			mainlog.With("mac", mac).Info("created hardware for mac: '" + mac.String() + "' with id: " + j.ID())
			mainlog.With("mac", mac).Info("Finding default template")
			// Hardware is now created, we must now grab the 'default' template
			tid, err := job.GetTemplate("default")
			if err != nil {
				mainlog.With("mac", mac).Error(err, "no default template exists")
				metrics.JobsInProgress.With(labels).Dec()
				timer.ObserveDuration()
				return
			}

			mainlog.With("mac", mac).Info("found default template with id: " + tid)
			mainlog.With("mac", mac).Info("Creating default workflow for machine: " + mac.String())
			// We have the 'default' template ID, now to make a workflow from it
			wid, err := job.CreateWorkflow(tid, mac)
			if err != nil {
				mainlog.With("mac", mac).Error(err, "failed to create workflow")
				metrics.JobsInProgress.With(labels).Dec()
				timer.ObserveDuration()
				return
			}
			mainlog.With("mac", mac).Info("created default workflow with id: " + wid)
		}
	}
	mainlog.With("mac", mac).Info("MAC address is known to tink")
	go func() {
		if j.ServeDHCP(w, req) {
			metrics.DHCPTotal.WithLabelValues("send", "DHCPOFFER", gi.String()).Inc()
		}
		metrics.JobsInProgress.With(labels).Dec()
		timer.ObserveDuration()
	}()
}

func getCircuitID(req *dhcp4.Packet) (string, error) {
	var circuitID string
	// Pulling option82 information from the packet (this is the relaying router)
	// format: byte 1 is option number, byte 2 is length of the following array of bytes.
	eightytwo, ok := req.GetOption(dhcp4.OptionRelayAgentInformation)
	if ok {
		if int(eightytwo[1]) < len(eightytwo) {
			circuitID = string(eightytwo[2:eightytwo[1]])
		} else {
			return circuitID, errors.New("option82 option1 out of bounds (check eightytwo[1])")
		}
	}
	return circuitID, nil
}
