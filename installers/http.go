package installers

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var httpHandlers = make(map[string]http.HandlerFunc)

func RegisterHTTPHandler(path string, fn http.HandlerFunc) {
	httpHandlers[path] = fn
}

func RegisterHTTPHandlers(mux *http.ServeMux) {
	for path, fn := range httpHandlers {
		mux.Handle(path, otelhttp.WithRouteTag(path, http.HandlerFunc(fn)))
	}
}
