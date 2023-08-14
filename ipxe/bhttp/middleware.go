package bhttp

import (
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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
	if log {
		h.log.V(1).Info("request", "method", method, "uri", uri, "client", client, "event", "sr")
	}

	res := &responseWriter{ResponseWriter: w}
	h.handler.ServeHTTP(res, req) // process the request

	if log {
		h.log.Info("response", "method", method, "uri", uri, "client", client, "duration", time.Since(start), "status", res.statusCode, "event", "ss")
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

	return n, errors.Wrap(err, "writing response")
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
