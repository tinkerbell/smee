package ipxe

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/dhcp/data"
	"github.com/tinkerbell/dhcp/handler"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ScriptHandler struct {
	Logger             logr.Logger
	Backend            handler.BackendReader
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
		d, n, err := h.Backend.GetByIP(ctx, ip)
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
		if !n.AllowNetboot {
			w.WriteHeader(http.StatusNotFound)
			h.Logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe; allow_pxe: false", "client", r.RemoteAddr)

			return
		}

		h.serveBootScript(ctx, w, d, n)
	}
}

func (h *ScriptHandler) serveBootScript(ctx context.Context, w http.ResponseWriter, d *data.DHCP, n *data.Netboot) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", "auto.ipxe"))
	s, err := h.defaultScript(span, d, n)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		err := fmt.Errorf("error generating ipxe script: %w", err)
		h.Logger.Error(err, "error generating ipxe script")
		span.SetStatus(codes.Error, err.Error())

		return
	}
	script := []byte(s)
	// check if the custom script should be used
	if cs, err := h.customScript(n); err == nil {
		script = []byte(cs)
	}
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		h.Logger.Error(errors.Wrap(err, "unable to send ipxe script"), "unable to send ipxe script", "client", d.IPAddress, "script", string(script))
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetStatus(codes.Ok, "boot script served")
}

func (h *ScriptHandler) defaultScript(span trace.Span, d *data.DHCP, n *data.Netboot) (string, error) {
	auto := Hook{
		Arch:              d.Arch,
		Console:           n.Console,
		DownloadURL:       h.OSIEURL,
		ExtraKernelParams: h.ExtraKernelParams,
		Facility:          n.Facility,
		HWAddr:            d.MACAddress.String(),
		SyslogHost:        h.PublicSyslogFQDN,
		TinkerbellTLS:     h.TinkServerTLS,
		TinkGRPCAuthority: h.TinkServerGRPCAddr,
		VLANID:            d.VLANID,
		WorkerID:          d.MACAddress.String(),
	}
	if sc := span.SpanContext(); sc.IsSampled() {
		auto.TraceID = sc.TraceID().String()
	}

	return GenerateTemplate(auto, HookScript)
}

// customScript returns the custom script or chain URL if defined in the hardware data otherwise an error.
func (h *ScriptHandler) customScript(n *data.Netboot) (string, error) {
	if n == nil {
		return "", errors.New("no hardware netboot data found")
	}
	if n.IPXEScriptURL != nil && n.IPXEScriptURL.String() != "" {
		c := Custom{Chain: n.IPXEScriptURL}
		return GenerateTemplate(c, CustomScript)
	}

	if script := n.IPXEScript; script != "" {
		c := Custom{Script: script}
		return GenerateTemplate(c, CustomScript)
	}

	return "", errors.New("no custom script or chain defined in the hardware data")
}
