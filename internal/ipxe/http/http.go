// package bhttp is the http server for smee.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/packethost/xff"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Config is the configuration for the http server.
type Config struct {
	GitRev         string
	StartTime      time.Time
	Logger         logr.Logger
	TrustedProxies []string
}

// HandlerMapping is a map of routes to http.HandlerFuncs.
type HandlerMapping map[string]http.HandlerFunc

// ServeHTTP sets up all the HTTP routes using a stdlib mux and starts the http
// server, which will block. App functionality is instrumented in Prometheus and OpenTelemetry.
func (s *Config) ServeHTTP(ctx context.Context, addr string, handlers HandlerMapping) error {
	mux := http.NewServeMux()
	for pattern, handler := range handlers {
		mux.Handle(otelFuncWrapper(pattern, handler))
	}

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthcheck", s.serveHealthchecker(s.GitRev, s.StartTime))

	// wrap the mux with an OpenTelemetry interceptor
	otelHandler := otelhttp.NewHandler(mux, "smee-http")

	// add X-Forwarded-For support if trusted proxies are configured
	var xffHandler http.Handler
	if len(s.TrustedProxies) > 0 {
		xffmw, err := xff.New(xff.Options{
			AllowedSubnets: s.TrustedProxies,
		})
		if err != nil {
			s.Logger.Error(err, "failed to create new xff object")
			panic(fmt.Errorf("failed to create new xff object: %v", err))
		}

		xffHandler = xffmw.Handler(&loggingMiddleware{
			handler: otelHandler,
			log:     s.Logger,
		})
	} else {
		xffHandler = &loggingMiddleware{
			handler: otelHandler,
			log:     s.Logger,
		}
	}

	server := http.Server{
		Addr:    addr,
		Handler: xffHandler,

		// Mitigate Slowloris attacks. 30 seconds is based on Apache's recommended 20-40
		// recommendation. Smee doesn't really have many headers so 20s should be plenty of time.
		// https://en.wikipedia.org/wiki/Slowloris_(computer_security)
		ReadHeaderTimeout: 20 * time.Second,
	}

	go func() {
		<-ctx.Done()
		s.Logger.Info("shutting down http server")
		_ = server.Shutdown(ctx)
	}()
	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		s.Logger.Error(err, "listen and serve http")
		return err
	}

	return nil
}

func (s *Config) serveHealthchecker(rev string, start time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
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
			s.Logger.Error(err, "marshaling healthcheck json")
		}
	}
}

// otelFuncWrapper takes a route and an http handler function, wraps the function
// with otelhttp, and returns the route again and http.Handler all set for mux.Handle().
func otelFuncWrapper(route string, h func(w http.ResponseWriter, req *http.Request)) (string, http.Handler) {
	return route, otelhttp.WithRouteTag(route, http.HandlerFunc(h))
}
