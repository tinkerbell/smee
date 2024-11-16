package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

type loggingMiddleware struct {
	handler http.Handler
	log     logr.Logger
}

// ServeHTTP implements http.Handler and add logging before and after the request.
func (h *loggingMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		start  = time.Now()
		method = req.Method
		uri    = req.RequestURI
		client = clientIP(req.RemoteAddr)
	)

	log := true
	if uri == "/metrics" {
		log = false
	}

	res := &responseWriter{ResponseWriter: w}
	h.handler.ServeHTTP(res, req) // process the request

	r := res.Header().Get("X-Global-Logging")

	if log && r == "" {
		h.log.Info("response", "method", method, "uri", uri, "client", client, "duration", time.Since(start), "status", res.statusCode)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		return 0, fmt.Errorf("failed writing response: %w", err)
	}

	return n, nil
}

func (w *responseWriter) WriteHeader(code int) {
	if w.statusCode == 0 {
		w.statusCode = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func clientIP(str string) string {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return "?"
	}

	return host
}
