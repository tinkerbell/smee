package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sebest/xff"
	"github.com/tinkerbell/boots/http/ipxe"
	"github.com/tinkerbell/dhcp/handler"
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
	Finder             handler.BackendReader
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
func (s *Config) ServeHTTP(ctx context.Context, addr string, ipxeBinaryHandler http.HandlerFunc) error {
	jh := ipxe.ScriptHandler{
		Logger:             s.Logger,
		Backend:            s.IPXEScript.Finder,
		OSIEURL:            s.IPXEScript.OsieURL,
		ExtraKernelParams:  s.IPXEScript.ExtraKernelParams,
		PublicSyslogFQDN:   s.IPXEScript.SyslogFQDN,
		TinkServerTLS:      s.IPXEScript.TinkServerTLS,
		TinkServerGRPCAddr: s.IPXEScript.TinkServerGRPCAddr,
	}
	mux := http.NewServeMux()
	mux.Handle(otelFuncWrapper("/auto.ipxe", jh.HandlerFunc()))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", s.serveHealthchecker(s.GitRev, s.StartTime))
	if ipxeBinaryHandler != nil {
		mux.Handle(otelFuncWrapper("/", ipxeBinaryHandler))
	}

	// wrap the mux with an OpenTelemetry interceptor
	otelHandler := otelhttp.NewHandler(mux, "boots-http")

	// add X-Forwarded-For support if trusted proxies are configured
	var bHandler http.Handler
	if len(s.TrustedProxies) > 0 {
		xffmw, err := xff.New(xff.Options{
			AllowedSubnets: s.TrustedProxies,
		})
		if err != nil {
			s.Logger.Error(err, "failed to create new xff object")
			return fmt.Errorf("failed to create new xff object: %w", err)
		}

		bHandler = xffmw.Handler(&loggingMiddleware{
			handler: otelHandler,
			log:     s.Logger,
		})
	} else {
		bHandler = &loggingMiddleware{
			handler: otelHandler,
			log:     s.Logger,
		}
	}

	srv := http.Server{
		Addr:    addr,
		Handler: bHandler,

		// Mitigate Slowloris attacks. 20 seconds is based on Apache's recommended 20-40
		// recommendation. Hegel doesn't really have many headers so 20s should be plenty of time.
		// https://en.wikipedia.org/wiki/Slowloris_(computer_security)
		ReadHeaderTimeout: 20 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait until we're told to shutdown.
	select {
	case <-ctx.Done():
	case e := <-errChan:
		return e
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt a graceful shutdown with timeout.
	//nolint:contextcheck // We can't derive from the original context as it's already done.
	if err := srv.Shutdown(ctx); err != nil {
		srv.Close()

		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("timed out waiting for graceful shutdown")
		}

		return err
	}

	return nil
}
