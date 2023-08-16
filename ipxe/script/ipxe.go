package script

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/dhcp/handler"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	Logger             logr.Logger
	Backend            handler.BackendReader
	OSIEURL            string
	ExtraKernelParams  []string
	PublicSyslogFQDN   string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
}

type Data struct {
	AllowNetboot  bool // If true, the client will be provided netboot options in the DHCP offer/ack.
	Console       string
	MACAddress    net.HardwareAddr
	Arch          string
	VLANID        string
	WorkflowID    string
	Facility      string
	IPXEScript    string
	IPXEScriptURL *url.URL
}

// Find implements the script.Finder interface.
// It uses the handler.BackendReader to get the (hardware) data and then
// translates it to the script.Data struct.
func GetByIP(ctx context.Context, ip net.IP, br handler.BackendReader) (Data, error) {
	d, n, err := br.GetByIP(ctx, ip)
	if err != nil {
		return Data{}, err
	}

	return Data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
	}, nil
}

type Finder interface {
	Find(context.Context, net.IP) (Data, error)
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
			h.Logger.Error(errors.Wrap(err, "splitting host:ip"), "error parsing client address", "client", r.RemoteAddr)

			return
		}
		ip := net.ParseIP(host)
		ctx := r.Context()
		// Should we serve a custom ipxe script?
		// This gates serving PXE file by
		// 1. the existence of a hardware record in tink server
		// AND
		// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
		// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
		// without a tink workflow present.
		hw, err := GetByIP(ctx, ip, h.Backend)
		if err != nil || !hw.AllowNetboot {
			w.WriteHeader(http.StatusNotFound)
			h.Logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe", "client", r.RemoteAddr, "error", err)

			return
		}

		h.serveBootScript(ctx, w, path.Base(r.URL.Path), hw)
	}
}

func (h *Handler) serveBootScript(ctx context.Context, w http.ResponseWriter, name string, hw Data) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("boots.script_name", name))
	var script []byte
	// check if the custom script should be used
	if hw.IPXEScriptURL != nil || hw.IPXEScript != "" {
		name = "custom.ipxe"
	}
	switch name {
	case "auto.ipxe":
		s, err := h.defaultScript(span, hw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err := errors.Wrap(err, "error with default ipxe script")
			h.Logger.Error(err, "error", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(s)
	case "custom.ipxe":
		cs, err := h.customScript(hw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err := errors.Wrap(err, "error with custom ipxe script")
			h.Logger.Error(err, "error", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(cs)
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
}

func (h *Handler) defaultScript(span trace.Span, hw Data) (string, error) {
	mac := hw.MACAddress
	arch := hw.Arch
	if arch == "" {
		arch = "x86_64"
	}
	// The worker ID will default to the mac address or use the one specified.
	wID := mac.String()
	if hw.WorkflowID != "" {
		wID = hw.WorkflowID
	}

	auto := Hook{
		Arch:              arch,
		Console:           "",
		DownloadURL:       h.OSIEURL,
		ExtraKernelParams: h.ExtraKernelParams,
		Facility:          hw.Facility,
		HWAddr:            mac.String(),
		SyslogHost:        h.PublicSyslogFQDN,
		TinkerbellTLS:     h.TinkServerTLS,
		TinkGRPCAuthority: h.TinkServerGRPCAddr,
		VLANID:            hw.VLANID,
		WorkerID:          wID,
	}
	if sc := span.SpanContext(); sc.IsSampled() {
		auto.TraceID = sc.TraceID().String()
	}

	return GenerateTemplate(auto, HookScript)
}

// customScript returns the custom script or chain URL if defined in the hardware data otherwise an error.
func (h *Handler) customScript(hw Data) (string, error) {
	if chain := hw.IPXEScriptURL; chain != nil && chain.String() != "" {
		if chain.Scheme != "http" && chain.Scheme != "https" {
			return "", fmt.Errorf("invalid URL scheme: %v", chain.Scheme)
		}
		c := Custom{Chain: chain}
		return GenerateTemplate(c, CustomScript)
	}
	if script := hw.IPXEScript; script != "" {
		c := Custom{Script: script}
		return GenerateTemplate(c, CustomScript)
	}

	return "", errors.New("no custom script or chain defined in the hardware data")
}
