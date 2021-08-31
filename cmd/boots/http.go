package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers/coreos"
	"github.com/tinkerbell/boots/installers/vmware"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	httpAddr = conf.HTTPBind
)

func init() {
	flag.StringVar(&httpAddr, "http-addr", httpAddr, "IP and port to listen on for HTTP.")
}

func serveHealthchecker(rev string, start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res := struct {
			GitRev     string  `json:"git_rev"`
			Uptime     float64 `json:"uptime"`
			Goroutines int     `json:"goroutines"`
		}{
			GitRev:     rev,
			Uptime:     time.Since(start).Seconds(),
			Goroutines: runtime.NumGoroutine(),
		}

		b, err := json.Marshal(&res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			mainlog.Error(errors.Wrap(err, "marshaling healtcheck json"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// otelFuncWrapper takes a route and an http handler function, wraps the function
// with otelhttp, and returns the route again and http.Handler all set for mux.Handle()
func otelFuncWrapper(route string, h func(w http.ResponseWriter, req *http.Request)) (string, http.Handler) {
	return route, otelhttp.WithRouteTag(route, http.HandlerFunc(h))
}

type jobHandler struct {
	i job.Installers
}

// ServeHTTP sets up all the HTTP routes using a stdlib mux and starts the http
// server, which will block. App functionality is instrumented in Prometheus and
// OpenTelemetry. Optionally configures X-Forwarded-For support.
func ServeHTTP(i job.Installers) {
	mux := http.NewServeMux()
	s := jobHandler{i: i}
	mux.Handle(otelFuncWrapper("/", s.serveJobFile))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/_packet/healthcheck", serveHealthchecker(GitRev, StartTime))
	mux.HandleFunc("/_packet/pprof/", pprof.Index)
	mux.HandleFunc("/_packet/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/_packet/pprof/profile", pprof.Profile)
	mux.HandleFunc("/_packet/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/_packet/pprof/trace", pprof.Trace)
	mux.HandleFunc("/healthcheck", serveHealthchecker(GitRev, StartTime))
	mux.Handle(otelFuncWrapper("/phone-home", servePhoneHome))
	mux.Handle(otelFuncWrapper("/phone-home/key", job.ServePublicKey))
	mux.Handle(otelFuncWrapper("/problem", serveProblem))
	mux.Handle(otelFuncWrapper("/hardware-components", serveHardware))

	// Events endpoint used to forward customer generated custom events from a running device (instance) to packet API
	mux.Handle(otelFuncWrapper("/events", func(w http.ResponseWriter, req *http.Request) {
		code, err := serveEvents(client, w, req)
		if err == nil {
			return
		}
		if code != http.StatusOK {
			mainlog.Error(err)
		}
	}))

	var httpHandlers = make(map[string]http.HandlerFunc)
	// register coreos/flatcar endpoints
	httpHandlers[coreos.IgnitionPathCoreos] = coreos.ServeIgnitionConfig("coreos")
	httpHandlers[coreos.IgnitionPathFlatcar] = coreos.ServeIgnitionConfig("flatcar")
	httpHandlers[coreos.OEMPath] = coreos.ServeOEM()
	// register vmware endpoints
	httpHandlers[vmware.KickstartPath] = vmware.ServeKickstart()

	// register Installer handlers
	for path, fn := range httpHandlers {
		mux.Handle(path, otelhttp.WithRouteTag(path, http.HandlerFunc(fn)))
	}

	// wrap the mux with an OpenTelemetry interceptor
	otelHandler := otelhttp.NewHandler(mux, "boots-http")

	// add X-Forwarded-For support if trusted proxies are configured
	var xffHandler http.Handler
	if len(conf.TrustedProxies) > 0 {
		xffmw, _ := xff.New(xff.Options{
			AllowedSubnets: conf.TrustedProxies,
		})

		xffHandler = xffmw.Handler(&httplog.Handler{
			Handler: otelHandler,
		})
	} else {
		xffHandler = &httplog.Handler{
			Handler: otelHandler,
		}
	}

	if err := http.ListenAndServe(httpAddr, xffHandler); err != nil {
		err = errors.Wrap(err, "listen and serve http")
		mainlog.Fatal(err)
	}
}

func (h *jobHandler) serveJobFile(w http.ResponseWriter, req *http.Request) {
	labels := prometheus.Labels{"from": "http", "op": "file"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	j, err := job.CreateFromRemoteAddr(req.Context(), req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")

		return
	}
	// This gates serving PXE file by
	// 1. the existence of a hardware record in tink server
	// AND
	// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
	// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
	// without a tink workflow present.
	if !j.AllowPxe() {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr).Info("the hardware data for this machine, or lack there of, does not allow it to pxe; allow_pxe: false")

		return
	}

	j.ServeFile(w, req, h.i)
}

func serveHardware(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	labels := prometheus.Labels{"from": "http", "op": "hardware-components"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	j, err := job.CreateFromRemoteAddr(ctx, req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")

		return
	}

	if j.CanWorkflow() {
		activeWorkflows, err := job.HasActiveWorkflow(ctx, j.HardwareID())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			j.With("error", err).Info("failed to get workflows")

			return
		}
		if !activeWorkflows {
			w.WriteHeader(http.StatusNotFound)
			j.Info("no active workflows")

			return
		}
	}

	j.AddHardware(w, req)
}

func servePhoneHome(w http.ResponseWriter, req *http.Request) {
	labels := prometheus.Labels{"from": "http", "op": "phone-home"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	j, err := job.CreateFromRemoteAddr(req.Context(), req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")

		return
	}
	j.ServePhoneHomeEndpoint(w, req)
}

func serveProblem(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	labels := prometheus.Labels{"from": "http", "op": "problem"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	j, err := job.CreateFromRemoteAddr(ctx, req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		mainlog.With("client", req.RemoteAddr, "error", err).Info("no job found for client address")

		return
	}

	if j.CanWorkflow() {
		activeWorkflows, err := job.HasActiveWorkflow(ctx, j.HardwareID())
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			j.With("error", err).Info("failed to get workflows")

			return
		}
		if !activeWorkflows {
			w.WriteHeader(http.StatusNotFound)
			j.Info("no active workflows")

			return
		}
	}

	j.ServeProblemEndpoint(w, req)
}

func readClose(r io.ReadCloser) (b []byte, err error) {
	b, err = ioutil.ReadAll(r)
	err = errors.Wrap(err, "read data")
	r.Close()

	return
}

type eventsServer interface {
	GetInstanceIDFromIP(context.Context, net.IP) (string, error)
	PostInstanceEvent(context.Context, string, io.Reader) (string, error)
}

// Forward user generated events to Packet API
func serveEvents(client eventsServer, w http.ResponseWriter, req *http.Request) (int, error) {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, errors.Wrap(err, "split host port")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		w.WriteHeader(http.StatusOK)

		return http.StatusOK, errors.New("no device found for client address")
	}

	deviceID, err := client.GetInstanceIDFromIP(req.Context(), ip)
	if err != nil || deviceID == "" {
		w.WriteHeader(http.StatusOK)

		return http.StatusOK, errors.New("no device found for client address")
	}

	b, err := readClose(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, err
	}
	if len(b) == 0 {
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, errors.New("userEvent body is empty")
	}

	var res struct {
		Code    int    `json:"code"`
		State   string `json:"state"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, errors.New("userEvent cannot be generated from supplied json")
	}

	e := struct {
		Code    string `json:"type"`
		State   string `json:"state"`
		Message string `json:"body"`
	}{
		Code:    "user." + strconv.Itoa(res.Code),
		State:   res.State,
		Message: res.Message,
	}
	payload, err := json.Marshal(e)
	if err != nil {
		// TODO(mmlb): this should be 500
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, errors.New("userEvent cannot be encoded")
	}

	if _, err := client.PostInstanceEvent(req.Context(), deviceID, bytes.NewReader(payload)); err != nil {
		// TODO(mmlb): this should be 500
		w.WriteHeader(http.StatusBadRequest)

		return http.StatusBadRequest, errors.New("failed to post userEvent")
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})

	return http.StatusOK, nil
}
