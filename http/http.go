package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/boots/http/ipxe"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Config struct {
	GitRev         string
	StartTime      time.Time
	Logger         logr.Logger
	TrustedProxies []string
	IPXEScript     *IPXEScript
}

type IPXEScript struct {
	Finder             backend.HardwareFinder
	Logger             logr.Logger
	OsieURL            string
	ExtraKernelParams  []string
	SyslogFQDN         string
	TinkServerTLS      bool
	TinkServerGRPCAddr string
}

func (s *Config) serveHealthchecker(rev string, start time.Time) http.HandlerFunc {
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
			s.Logger.Error(errors.Wrap(err, "marshaling healtcheck json"), "marshaling healtcheck json")
		}
	}
}

// otelFuncWrapper takes a route and an http handler function, wraps the function
// with otelhttp, and returns the route again and http.Handler all set for mux.Handle().
func otelFuncWrapper(route string, h func(w http.ResponseWriter, req *http.Request)) (string, http.Handler) {
	return route, otelhttp.WithRouteTag(route, http.HandlerFunc(h))
}

// ServeHTTP sets up all the HTTP routes using a stdlib mux and starts the http
// server, which will block. App functionality is instrumented in Prometheus and
// OpenTelemetry. Optionally configures X-Forwarded-For support.
func (s *Config) ServeHTTP(srv *http.Server, addr string, ipxePattern string, ipxeBinaryHandler http.HandlerFunc) error {
	jh := ipxe.ScriptHandler{
		Logger:             s.Logger,
		Finder:             s.IPXEScript.Finder,
		OSIEURL:            s.IPXEScript.OsieURL,
		ExtraKernelParams:  s.IPXEScript.ExtraKernelParams,
		PublicSyslogFQDN:   s.IPXEScript.SyslogFQDN,
		TinkServerTLS:      s.IPXEScript.TinkServerTLS,
		TinkServerGRPCAddr: s.IPXEScript.TinkServerGRPCAddr,
	}
	mux := http.NewServeMux()
	mux.Handle(otelFuncWrapper("/", jh.HandlerFunc()))
	if ipxeBinaryHandler != nil {
		mux.Handle(otelFuncWrapper(ipxePattern, ipxeBinaryHandler))
	}
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthcheck", s.serveHealthchecker(s.GitRev, s.StartTime))

	// wrap the mux with an OpenTelemetry interceptor
	otelHandler := otelhttp.NewHandler(mux, "boots-http")

	// add X-Forwarded-For support if trusted proxies are configured
	var xffHandler http.Handler
	if len(s.TrustedProxies) > 0 {
		xffmw, err := xff.New(xff.Options{
			AllowedSubnets: s.TrustedProxies,
		})
		if err != nil {
			s.Logger.Error(err, "failed to create new xff object")
			return fmt.Errorf("failed to create new xff object: %w", err)
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

	srv.Addr = addr
	srv.Handler = xffHandler
	// Mitigate Slowloris attacks. 30 seconds is based on Apache's recommended 20-40
	// recommendation. Boots doesn't really have many headers so 20s should be plenty of time.
	// https://en.wikipedia.org/wiki/Slowloris_(computer_security)
	srv.ReadHeaderTimeout = 20 * time.Second

	return srv.ListenAndServe()
}
