package main

import (
	"context"
	"flag"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

	tracer := otel.Tracer("DHCP")
	ctx, span := tracer.Start(context.Background(), "ServeDHCP",
		trace.WithAttributes(attribute.String("MAC", mac.String())),
		trace.WithAttributes(attribute.String("IP", gi.String())),
		trace.WithAttributes(attribute.String("MessageType", req.GetMessageType().String())),
		trace.WithAttributes(attribute.String("CircuitID", circuitID)),
	)

	j, err := job.CreateFromDHCP(ctx, mac, gi, circuitID)
	if err != nil {
		mainlog.With("type", req.GetMessageType(), "mac", mac, "err", err).Info("retrieved job is empty")
		metrics.JobsInProgress.With(labels).Dec()
		timer.ObserveDuration()
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return
	}
	span.End()

	go func() {
		ctx, span := tracer.Start(ctx, "ServeDHCP Reply")
		ok, err := j.ServeDHCP(ctx, w, req)
		if ok {
			span.SetStatus(codes.Ok, "DHCPOFFER sent")
			metrics.DHCPTotal.WithLabelValues("send", "DHCPOFFER", gi.String()).Inc()
		} else {
			if err != nil {
				j.Error(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "no offer made")
			}
		}
		span.End()
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
