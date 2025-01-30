package script

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
	"github.com/tinkerbell/smee/internal/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	Logger                logr.Logger
	Backend               handler.BackendReader
	OSIEURL               string
	ExtraKernelParams     []string
	PublicSyslogFQDN      string
	TinkServerTLS         bool
	TinkServerInsecureTLS bool
	TinkServerGRPCAddr    string
	IPXEScriptRetries     int
	IPXEScriptRetryDelay  int
	StaticIPXEEnabled     bool
}

type data struct {
	AllowNetboot  bool // If true, the client will be provided netboot options in the DHCP offer/ack.
	Console       string
	MACAddress    net.HardwareAddr
	Arch          string
	VLANID        string
	WorkflowID    string
	Facility      string
	IPXEScript    string
	IPXEScriptURL *url.URL
	OSIE          OSIE
}

// OSIE or OS Installation Environment is the data about where the OSIE parts are located.
type OSIE struct {
	// BaseURL is the URL where the OSIE parts are located.
	BaseURL *url.URL
	// Kernel is the name of the kernel file.
	Kernel string
	// Initrd is the name of the initrd file.
	Initrd string
}

// getByMac uses the handler.BackendReader to get the (hardware) data and then
// translates it to the script.Data struct.
func getByMac(ctx context.Context, mac net.HardwareAddr, br handler.BackendReader) (data, error) {
	if br == nil {
		return data{}, errors.New("backend is nil")
	}
	d, n, err := br.GetByMac(ctx, mac)
	if err != nil {
		return data{}, err
	}

	return data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
		OSIE:          OSIE(n.OSIE),
	}, nil
}

func getByIP(ctx context.Context, ip net.IP, br handler.BackendReader) (data, error) {
	if br == nil {
		return data{}, errors.New("backend is nil")
	}
	d, n, err := br.GetByIP(ctx, ip)
	if err != nil {
		return data{}, err
	}

	return data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
		OSIE:          OSIE(n.OSIE),
	}, nil
}

// HandlerFunc returns a http.HandlerFunc that serves the ipxe script.
// It is expected that the request path is /<mac address>/auto.ipxe.
func (h *Handler) HandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if path.Base(r.URL.Path) != "auto.ipxe" {
			h.Logger.Info("URL path not supported", "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}
		labels := prometheus.Labels{"from": "http", "op": "file"}
		metric.JobsTotal.With(labels).Inc()
		metric.JobsInProgress.With(labels).Inc()
		defer metric.JobsInProgress.With(labels).Dec()
		timer := prometheus.NewTimer(metric.JobDuration.With(labels))
		defer timer.ObserveDuration()

		ctx := r.Context()

		// Should we serve a custom ipxe script?
		// This gates serving PXE file by
		// 1. the existence of a hardware record in tink server
		// AND
		// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
		// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
		// without a tink workflow present.

		// Try to get the MAC address from the URL path, if not available get the source IP address.
		if ha, err := getMAC(r.URL.Path); err == nil {
			hw, err := getByMac(ctx, ha, h.Backend)
			if err != nil && h.StaticIPXEEnabled {
				h.Logger.Info("serving static ipxe script", "mac", ha, "error", err)
				h.serveStaticIPXEScript(w)
				return
			}
			if err != nil || !hw.AllowNetboot {
				w.WriteHeader(http.StatusNotFound)
				h.Logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe", "client", ha, "error", err)

				return
			}
			h.serveBootScript(ctx, w, path.Base(r.URL.Path), hw)
			return
		}
		if ip, err := getIP(r.RemoteAddr); err == nil {
			hw, err := getByIP(ctx, ip, h.Backend)
			if err != nil && h.StaticIPXEEnabled {
				h.Logger.Info("serving static ipxe script", "client", r.RemoteAddr, "error", err)
				h.serveStaticIPXEScript(w)
				return
			}
			if err != nil || !hw.AllowNetboot {
				w.WriteHeader(http.StatusNotFound)
				h.Logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe", "client", r.RemoteAddr, "error", err)

				return
			}
			h.serveBootScript(ctx, w, path.Base(r.URL.Path), hw)
			return
		}

		// If we get here, we were unable to get the MAC address from the URL path or the source IP address.
		w.WriteHeader(http.StatusNotFound)
		h.Logger.Info("unable to get the MAC address from the URL path or the source IP address", "client", r.RemoteAddr, "urlPath", r.URL.Path)
	}
}

func (h *Handler) serveStaticIPXEScript(w http.ResponseWriter) {
	// Serve static iPXE script.
	auto := Hook{
		DownloadURL:       h.OSIEURL,
		ExtraKernelParams: h.ExtraKernelParams,
		SyslogHost:        h.PublicSyslogFQDN,
		TinkerbellTLS:     h.TinkServerTLS,
		TinkGRPCAuthority: h.TinkServerGRPCAddr,
	}
	script, err := GenerateTemplate(auto, StaticScript)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Error(err, "error generating the static ipxe script")
		return
	}
	if _, err := w.Write([]byte(script)); err != nil {
		h.Logger.Error(err, "unable to send the static ipxe script")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func getIP(remoteAddr string) (net.IP, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.IP{}, fmt.Errorf("error parsing client address: %w: client: %v", err, remoteAddr)
	}
	ip := net.ParseIP(host)

	return ip, nil
}

func getMAC(urlPath string) (net.HardwareAddr, error) {
	mac := path.Base(path.Dir(urlPath))
	ha, err := net.ParseMAC(mac)
	if err != nil {
		return net.HardwareAddr{}, fmt.Errorf("URL path not supported, the second to last element in the URL path must be a valid mac address, err: %w", err)
	}

	return ha, nil
}

func (h *Handler) serveBootScript(ctx context.Context, w http.ResponseWriter, name string, hw data) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("smee.script_name", name))
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
			h.Logger.Error(err, "error with default ipxe script", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(s)
	case "custom.ipxe":
		cs, err := h.customScript(hw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Error(err, "error with custom ipxe script", "script", name)
			span.SetStatus(codes.Error, err.Error())

			return
		}
		script = []byte(cs)
	default:
		w.WriteHeader(http.StatusNotFound)
		err := fmt.Errorf("boot script %q not found", name)
		h.Logger.Error(err, "boot script not found", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
	span.SetAttributes(attribute.String("ipxe-script", string(script)))

	if _, err := w.Write(script); err != nil {
		h.Logger.Error(err, "unable to write boot script", "script", name)
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

func (h *Handler) defaultScript(span trace.Span, hw data) (string, error) {
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
		Arch:                  arch,
		Console:               "",
		DownloadURL:           h.OSIEURL,
		ExtraKernelParams:     h.ExtraKernelParams,
		Facility:              hw.Facility,
		HWAddr:                mac.String(),
		SyslogHost:            h.PublicSyslogFQDN,
		TinkerbellTLS:         h.TinkServerTLS,
		TinkerbellInsecureTLS: h.TinkServerInsecureTLS,
		TinkGRPCAuthority:     h.TinkServerGRPCAddr,
		VLANID:                hw.VLANID,
		WorkerID:              wID,
		Retries:               h.IPXEScriptRetries,
		RetryDelay:            h.IPXEScriptRetryDelay,
	}
	if hw.OSIE.BaseURL != nil && hw.OSIE.BaseURL.String() != "" {
		auto.DownloadURL = hw.OSIE.BaseURL.String()
	}
	if hw.OSIE.Kernel != "" {
		auto.Kernel = hw.OSIE.Kernel
	}
	if hw.OSIE.Initrd != "" {
		auto.Initrd = hw.OSIE.Initrd
	}
	if sc := span.SpanContext(); sc.IsSampled() {
		auto.TraceID = sc.TraceID().String()
	}

	return GenerateTemplate(auto, HookScript)
}

// customScript returns the custom script or chain URL if defined in the hardware data otherwise an error.
func (h *Handler) customScript(hw data) (string, error) {
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
