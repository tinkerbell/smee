package job

import (
	"net/http"
)

// AddHardware - Add hardware component(s).
func (j Job) AddHardware(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte{})
}
