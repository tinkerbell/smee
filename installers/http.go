package installers

import (
	"net/http"
)

var httpHandlers = make(map[string]http.HandlerFunc)

func RegisterHTTPHandler(path string, fn http.HandlerFunc) {
	httpHandlers[path] = fn
}

func RegisterHTTPHandlers(mux *http.ServeMux) {
	for path, fn := range httpHandlers {
		mux.HandleFunc(path, fn)
	}
}
