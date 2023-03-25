package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type BootsHTTPServer struct {
	finder     client.HardwareFinder
	jobManager job.Manager
	logger     logr.Logger
}

func (s *BootsHTTPServer) serveHealthchecker(rev string, start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res := struct {
			GitRev     string  `json:"git_rev"`
			Uptime     float64 `json:"uptime"`
			Goroutines int     `json:"goroutines"`
		}{
			GitRev:     rev,
			Uptime:     time.Since(start).Seconds(),
			Goroutines: runtime.NumGoroutine(),
		}
		if err := json.NewEncoder(w).Encode(&res); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.logger.Error(errors.Wrap(err, "marshaling healtcheck json"), "marshaling healtcheck json")
		}
	}
}

// otelFuncWrapper takes a route and an http handler function, wraps the function
// with otelhttp, and returns the route again and http.Handler all set for mux.Handle().
func otelFuncWrapper(route string, h func(w http.ResponseWriter, req *http.Request)) (string, http.Handler) {
	return route, otelhttp.WithRouteTag(route, http.HandlerFunc(h))
}

type jobHandler struct {
	jobManager job.Manager
	logger     logr.Logger
}

// ServeHTTP sets up all the HTTP routes using a stdlib mux and starts the http
// server, which will block. App functionality is instrumented in Prometheus and
// OpenTelemetry. Optionally configures X-Forwarded-For support.
func (s *BootsHTTPServer) ServeHTTP(addr string, ipxePattern string, ipxeHandler func(http.ResponseWriter, *http.Request)) {
	mux := http.NewServeMux()
	jh := jobHandler{jobManager: s.jobManager, logger: s.logger}
	mux.Handle(otelFuncWrapper("/", jh.serveJobFile))
	if ipxeHandler != nil {
		mux.Handle(otelFuncWrapper(ipxePattern, ipxeHandler))
	}
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/_packet/healthcheck", s.serveHealthchecker(GitRev, StartTime))
	mux.HandleFunc("/_packet/pprof/", pprof.Index)
	mux.HandleFunc("/_packet/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/_packet/pprof/profile", pprof.Profile)
	mux.HandleFunc("/_packet/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/_packet/pprof/trace", pprof.Trace)
	mux.HandleFunc("/healthcheck", s.serveHealthchecker(GitRev, StartTime))
	mux.Handle(otelFuncWrapper("/phone-home", s.servePhoneHome))

	// wrap the mux with an OpenTelemetry interceptor
	otelHandler := otelhttp.NewHandler(mux, "boots-http")

	// add X-Forwarded-For support if trusted proxies are configured
	var xffHandler http.Handler
	if len(conf.TrustedProxies) > 0 {
		xffmw, err := xff.New(xff.Options{
			AllowedSubnets: conf.TrustedProxies,
		})
		if err != nil {
			s.logger.Error(err, "failed to create new xff object")
			panic(fmt.Errorf("failed to create new xff object: %v", err))
		}

		xffHandler = xffmw.Handler(&httplog.Handler{
			Handler: otelHandler,
			Log:     s.logger,
		})
	} else {
		xffHandler = &httplog.Handler{
			Handler: otelHandler,
			Log:     s.logger,
		}
	}

	server := http.Server{
		Addr:    addr,
		Handler: xffHandler,

		// Mitigate Slowloris attacks. 30 seconds is based on Apache's recommended 20-40
		// recommendation. Boots doesn't really have many headers so 20s should be plenty of time.
		// https://en.wikipedia.org/wiki/Slowloris_(computer_security)
		ReadHeaderTimeout: 20 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		err = errors.Wrap(err, "listen and serve http")
		s.logger.Error(err, "listen and serve http")
		panic(err)
	}
}

func (h *jobHandler) serveJobFile(w http.ResponseWriter, req *http.Request) {
	labels := prometheus.Labels{"from": "http", "op": "file"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	ctx, j, err := h.jobManager.CreateFromRemoteAddr(req.Context(), req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		h.logger.Error(err, "no job found for client address", "client", req.RemoteAddr)

		return
	}
	// This gates serving PXE file by
	// 1. the existence of a hardware record in tink server
	// AND
	// 2. the network.interfaces[].netboot.allow_pxe value, in the tink server hardware record, equal to true
	// This allows serving custom ipxe scripts, starting up into OSIE or other installation environments
	// without a tink workflow present.
	if !j.AllowPXE() {
		w.WriteHeader(http.StatusNotFound)
		h.logger.Info("the hardware data for this machine, or lack there of, does not allow it to pxe; allow_pxe: false", "client", req.RemoteAddr)

		return
	}

	// otel: send a req.Clone with the updated context from the job's hw data
	j.ServeFile(w, req.Clone(ctx))
}

func (s *BootsHTTPServer) servePhoneHome(w http.ResponseWriter, req *http.Request) {
	labels := prometheus.Labels{"from": "http", "op": "phone-home"}
	metrics.JobsTotal.With(labels).Inc()
	metrics.JobsInProgress.With(labels).Inc()
	defer metrics.JobsInProgress.With(labels).Dec()
	timer := prometheus.NewTimer(metrics.JobDuration.With(labels))
	defer timer.ObserveDuration()

	_, j, err := s.jobManager.CreateFromRemoteAddr(req.Context(), req.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		s.logger.Info("no job found for client address", "client", req.RemoteAddr, "error", err)

		return
	}
	j.ServePhoneHomeEndpoint(w, req)
}
