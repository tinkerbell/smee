package ipxe

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	Logger             logr.Logger
	Finder             client.HardwareFinder
	OSIEURL            string
	ExtraKernelParams  []string
	PublicSyslogFQDN   string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
}

func (h *Handler) HandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if path.Base(r.URL.Path) != "auto.ipxe" {
			h.Logger.Info("not found", "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}
		labels := prometheus.Labels{"from": "http", "op": "file"}
		metrics.JobsTotal.With(labels).Inc()
		metrics.JobsInProgress.With(labels).Inc()
		defer metrics.JobsInProgress.With(labels).Dec()
		timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
		defer timer.ObserveDuration()

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Info("unable to parse client address", "client", r.RemoteAddr)

			return
		}
		ip := net.ParseIP(host)
		ctx := r.Context()
		// get hardware record
		hw, err := h.Finder.ByIP(ctx, ip)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			h.Logger.Info("no job found for client address", "client", ip, "details", err.Error())

			return
		}
		// This gates serving PXE file by
		// 1. the existence of a hardware record in tink server
		// AND
		// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
		// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
		// without a tink workflow present.
		if !hw.Hardware().HardwareAllowPXE(hw.GetMAC(ip)) {
			w.WriteHeader(http.StatusNotFound)
			h.Logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe; allow_pxe: false", "client", r.RemoteAddr)

			return
		}

		h.serveBootScript(ctx, w, path.Base(r.URL.Path), ip.String(), hw)
	}
}

func (h *Handler) serveBootScript(ctx context.Context, w http.ResponseWriter, name string, ip string, hw client.Discoverer) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", name))
	var script []byte
	switch name {
	case "auto.ipxe":
		// check if the custom script should be used
		if cs, err := h.customScript(hw, ip); err == nil {
			script = []byte(cs)
			break
		}
		s, err := h.defaultScript(span, hw, ip)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			err := errors.Errorf("boot script %q not found", name)
			h.Logger.Error(err, "error", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(s)
	default:
		w.WriteHeader(http.StatusNotFound)
		err := errors.Errorf("boot script %q not found", name)
		h.Logger.Error(err, "error", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		h.Logger.Error(errors.Wrap(err, "unable to write boot script"), "unable to write boot script", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetStatus(codes.Ok, "boot script served")
}

func (h *Handler) defaultScript(span trace.Span, hw client.Discoverer, ip string) (string, error) {
	auto := Hook{
		Arch:              hw.Hardware().HardwareArch(hw.GetMAC(net.ParseIP(ip))),
		Console:           "",
		DownloadURL:       h.OSIEURL,
		ExtraKernelParams: h.ExtraKernelParams,
		Facility:          hw.Hardware().HardwareFacilityCode(),
		HWAddr:            hw.GetMAC(net.ParseIP(ip)).String(),
		SyslogHost:        h.PublicSyslogFQDN,
		TinkerbellTLS:     h.TinkServerTLS,
		TinkGRPCAuthority: h.TinkServerGRPCAddr,
		VLANID:            hw.Hardware().GetVLANID(hw.GetMAC(net.ParseIP(ip))),
		WorkerID:          hw.Instance().ID,
	}
	if sc := span.SpanContext(); sc.IsSampled() {
		auto.TraceID = sc.TraceID().String()
	}

	return GenerateTemplate(auto, HookScript)
}

// customScript returns the custom script or chain URL if defined in the hardware data otherwise an error.
func (h *Handler) customScript(hw client.Discoverer, ip string) (string, error) {
	mac := hw.GetMAC(net.ParseIP(ip))
	if chain := hw.Hardware().IPXEURL(mac); chain != "" {
		u, err := url.Parse(chain)
		if err != nil {
			return "", errors.Wrap(err, "invalid custom chain URL")
		}
		c := Custom{Chain: u}
		return GenerateTemplate(c, CustomScript)
	}
	if script := hw.Hardware().IPXEScript(mac); script != "" {
		c := Custom{Script: script}
		return GenerateTemplate(c, CustomScript)
	}

	return "", errors.New("no custom script or chain defined in the hardware data")
}
