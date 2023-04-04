package http

import (
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
		logger = h.log.WithValues("method", method, "uri", uri, "client", client)
	)

	if uri != "/metrics" && uri != "/healthz" {
		logger.V(1).Info("request")
	}

	recorder := &statusRecorder{
		ResponseWriter: w,
		Status:         200,
	}
	h.handler.ServeHTTP(recorder, req) // process the request

	if uri != "/metrics" && uri != "/healthz" {
		logger.V(1).Info("response", "duration", time.Since(start), "statusCode", recorder.Status)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.Status = status
	s.ResponseWriter.WriteHeader(status)
}

func clientIP(str string) string {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return "?"
	}

	return host
}
