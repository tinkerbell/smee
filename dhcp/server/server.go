package server

import (
	"context"
	"net"
	"strings"

	"github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	"github.com/go-logr/logr"
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	JobManager Manager
	Logger     logr.Logger
	PoolSize   int
}

// Manager creates jobs.
type Manager interface {
	CreateFromDHCP(context.Context, net.HardwareAddr, net.IP, string) (context.Context, *job.Job, error)
}

// ServeDHCP starts the DHCP server.
// It takes the next server address (nextServer) for serving iPXE binaries via TFTP
// and an IP:Port (httpServerFQDN) for serving iPXE binaries via HTTP.
func (h *Handler) ServeDHCP(addr string, nextServer net.IP, ipxeBaseURL string, bootsBaseURL string) {
	handler := dhcpHandler{
		pool:         workerpool.New(h.PoolSize),
		nextServer:   nextServer,
		ipxeBaseURL:  ipxeBaseURL,
		bootsBaseURL: bootsBaseURL,
		jobManager:   h.JobManager,
		logger:       h.Logger,
	}
	defer handler.pool.Stop()

	err := retry.Do(
		func() error {
			return errors.Wrap(dhcp4.ListenAndServe(addr, handler), "serving dhcp")
		},
	)
	if err != nil {
		h.Logger.Error(err, "failed to serve dhcp")
		panic(errors.Wrap(err, "retry dhcp serve"))
	}
}

type dhcpHandler struct {
	pool         *workerpool.WorkerPool
	nextServer   net.IP
	ipxeBaseURL  string
	bootsBaseURL string
	jobManager   Manager
	logger       logr.Logger
	ignoredMACS  string
	ignoredIPs   string
}

func (d dhcpHandler) ServeDHCP(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	d.pool.Submit(func() { d.serve(w, req) })
}

func (d dhcpHandler) shouldIgnoreOUI(mac string) bool {
	ignoredOUIs := d.getIgnoredMACs()
	if ignoredOUIs == nil {
		return false
	}
	oui := strings.ToLower(mac[:8])
	_, ok := ignoredOUIs[oui]

	return ok
}

func (d dhcpHandler) getIgnoredMACs() map[string]struct{} {
	macs := d.ignoredMACS
	if macs == "" {
		return nil
	}

	slice := strings.Split(macs, ",")
	if len(slice) == 0 {
		return nil
	}

	ignore := map[string]struct{}{}
	for _, oui := range slice {
		_, err := net.ParseMAC(oui + ":00:00:00")
		if err != nil {
			panic(errors.Errorf("invalid oui in TINK_IGNORED_OUIS oui=%s", oui))
		}
		oui = strings.ToLower(oui)
		ignore[oui] = struct{}{}
	}

	return ignore
}

func (d dhcpHandler) shouldIgnoreGI(ip string) bool {
	ignoredGIs := d.getIgnoredGIs()
	if ignoredGIs == nil {
		return false
	}
	_, ok := ignoredGIs[ip]

	return ok
}

func (d dhcpHandler) getIgnoredGIs() map[string]struct{} {
	ips := d.ignoredIPs
	if ips == "" {
		return nil
	}

	slice := strings.Split(ips, ",")
	if len(slice) == 0 {
		return nil
	}

	ignore := map[string]struct{}{}
	for _, ip := range slice {
		if net.ParseIP(ip) == nil {
			panic(errors.Errorf("invalid ip address in TINK_IGNORED_GIS ip=%s", ip))
		}
		ignore[ip] = struct{}{}
	}

	return ignore
}

func (d dhcpHandler) serve(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	mac := req.GetCHAddr()
	if d.shouldIgnoreOUI(mac.String()) {
		d.logger.Info("mac is in ignore list", "mac", mac)

		return
	}

	gi := req.GetGIAddr()
	if d.shouldIgnoreGI(gi.String()) {
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

	ctx, j, err := d.jobManager.CreateFromDHCP(ctx, mac, gi, circuitID)
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
