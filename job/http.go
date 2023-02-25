package job

import (
	"net/http"
	"path"
	"strings"
)

func (j Job) ServeFile(w http.ResponseWriter, req *http.Request) {
	base := path.Base(req.URL.Path)

	if name := strings.TrimSuffix(base, ".ipxe"); len(name) < len(base) {
		j.serveBootScript(req.Context(), w, name)

		return
	}
}

func (j Job) ServePhoneHomeEndpoint(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte{})
}
