package httplog

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Handler struct {
	http.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var (
		start  = time.Now()
		method = req.Method
		uri    = req.RequestURI
		client = clientIP(req.RemoteAddr)
	)

	log := true
	if uri == "/metrics" || strings.HasPrefix(uri, "/_packet") {
		log = false
	}
	if log {
		httplog.With("event", "sr", "method", method, "uri", uri, "client", client).Debug()
	}

	res := &ResponseWriter{ResponseWriter: w}
	h.Handler.ServeHTTP(res, req) // process the request
	d := time.Since(start)

	if log {
		httplog.With("event", "ss", "method", method, "uri", uri, "client", client, "duration", d, "status", res.StatusCode).Info()
	}
}

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	if w.StatusCode == 0 {
		w.StatusCode = 200
	}
	n, err := w.ResponseWriter.Write(b)

	return n, errors.Wrap(err, "writing response")
}

func (w *ResponseWriter) WriteHeader(code int) {
	if w.StatusCode == 0 {
		w.StatusCode = code
	}
	w.ResponseWriter.WriteHeader(code)
}

type Transport struct {
	http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var (
		method = req.Method
		uri    = req.URL.String()
	)
	httplog.With("event", "cs", "method", method, "uri", uri).Debug()

	start := time.Now()
	res, err = t.RoundTripper.RoundTrip(req)
	d := time.Since(start)

	if res != nil {
		httplog.With("event", "cr", "method", method, "uri", uri, "duration", d, "status", res.StatusCode).Info()
	}

	return
}

func clientIP(str string) string {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return "?"
	}

	return host
}
