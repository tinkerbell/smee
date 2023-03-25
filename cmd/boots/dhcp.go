package main

import (
	"context"
	"net"
	"runtime"

	"github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	"github.com/go-logr/logr"
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

type BootsDHCPServer struct {
	jobmanager job.Manager

	Logger logr.Logger
}

// ServeDHCP starts the DHCP server.
// It takes the next server address (nextServer) for serving iPXE binaries via TFTP
// and an IP:Port (httpServerFQDN) for serving iPXE binaries via HTTP.
func (s *BootsDHCPServer) ServeDHCP(addr string, nextServer net.IP, ipxeBaseURL string, bootsBaseURL string) {
	poolSize := env.Int("BOOTS_DHCP_WORKERS", runtime.GOMAXPROCS(0)/2)
	handler := dhcpHandler{
		pool:         workerpool.New(poolSize),
		nextServer:   nextServer,
		ipxeBaseURL:  ipxeBaseURL,
		bootsBaseURL: bootsBaseURL,
		jobmanager:   s.jobmanager,
		logger:       s.Logger,
	}
	defer handler.pool.Stop()

	err := retry.Do(
		func() error {
			return errors.Wrap(dhcp4.ListenAndServe(addr, handler), "serving dhcp")
		},
	)
	if err != nil {
		panic(errors.Wrap(err, "retry dhcp serve"))
	}
}

type dhcpHandler struct {
	pool         *workerpool.WorkerPool
	nextServer   net.IP
	ipxeBaseURL  string
	bootsBaseURL string
	jobmanager   job.Manager
	logger       logr.Logger
}

func (d dhcpHandler) ServeDHCP(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	d.pool.Submit(func() { d.serve(w, req) })
}

func (d dhcpHandler) serve(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	mac := req.GetCHAddr()
	if conf.ShouldIgnoreOUI(mac.String()) {
		d.logger.Info("mac is in ignore list", "mac", mac)

		return
	}

	gi := req.GetGIAddr()
	if conf.ShouldIgnoreGI(gi.String()) {
		d.logger.Info("giaddr is in ignore list", "giaddr", gi)

		return
	}

	metrics.DHCPTotal.WithLabelValues("recv", req.GetMessageType().String(), gi.String()).Inc()
	labels := prometheus.Labels{"from": "dhcp", "op": req.GetMessageType().String()}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))

	circuitID, err := getCircuitID(req)
	if err != nil {
		d.logger.Error(err, "error parsing option82", "mac", mac)
	} else {
		d.logger.Info("parsed option82/circuitid", "mac", mac, "circuitID", circuitID)
	}

	tracer := otel.Tracer("DHCP")
	ctx, span := tracer.Start(context.Background(), "DHCP Reply",
		trace.WithAttributes(attribute.String("MAC", mac.String())),
		trace.WithAttributes(attribute.String("IP", gi.String())),
		trace.WithAttributes(attribute.String("MessageType", req.GetMessageType().String())),
		trace.WithAttributes(attribute.String("CircuitID", circuitID)),
	)

	ctx, j, err := d.jobmanager.CreateFromDHCP(ctx, mac, gi, circuitID)
	if err != nil {
		d.logger.Error(err, "retrieved job is empty", "type", req.GetMessageType(), "mac", mac)
		metrics.JobsInProgress.With(labels).Dec()
		timer.ObserveDuration()
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return
	}
	span.End()
	j.IpxeBaseURL = d.ipxeBaseURL
	j.BootsBaseURL = d.bootsBaseURL
	j.NextServer = d.nextServer

	go func() {
		ctx, span := tracer.Start(ctx, "DHCP Reply")
		ok, err := j.ServeDHCP(ctx, w, req)
		if ok {
			span.SetStatus(codes.Ok, "DHCPOFFER sent")
			metrics.DHCPTotal.WithLabelValues("send", "DHCPOFFER", gi.String()).Inc()
		} else {
			if err != nil {
				d.logger.Error(err, "error")
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
