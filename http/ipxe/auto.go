package ipxe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/boots/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ScriptHandler struct {
	Logger             logr.Logger
	Finder             backend.HardwareFinder
	OSIEURL            string
	ExtraKernelParams  []string
	PublicSyslogFQDN   string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
}

func (h *ScriptHandler) HandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		h.serveBootScript(ctx, w, ip.String(), hw)
	}
}

func (h *ScriptHandler) serveBootScript(ctx context.Context, w http.ResponseWriter, ip string, hw backend.Discoverer) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", "auto.ipxe"))
	s, err := h.defaultScript(span, hw, ip)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		err := fmt.Errorf("error generating ipxe script: %w", err)
		h.Logger.Error(err, "error generating ipxe script")
		span.SetStatus(codes.Error, err.Error())

		return
	}
	script := []byte(s)
	// check if the custom script should be used
	if cs, err := h.customScript(hw, ip); err == nil {
		script = []byte(cs)
	}
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		h.Logger.Error(errors.Wrap(err, "unable to send ipxe script"), "unable to send ipxe script", "client", ip, "script", string(script))
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetStatus(codes.Ok, "boot script served")
}

func (h *ScriptHandler) defaultScript(span trace.Span, hw backend.Discoverer, ip string) (string, error) {
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
func (h *ScriptHandler) customScript(hw backend.Discoverer, ip string) (string, error) {
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
