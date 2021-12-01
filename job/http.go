package job

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/pkg/errors"
)

func (j Job) ServeFile(w http.ResponseWriter, req *http.Request, i Installers) {
	base := path.Base(req.URL.Path)

	if name := strings.TrimSuffix(base, ".ipxe"); len(name) < len(base) {
		j.serveBootScript(req.Context(), w, name, i)

		return
	}

	w.WriteHeader(http.StatusNotFound)
	j.With("file", base).Info("file not found")
}

func (j Job) ServePhoneHomeEndpoint(w http.ResponseWriter, req *http.Request) {
	var b []byte

	switch req.Header.Get("Content-Type") {
	case "application/json", "":
		// cloudbase-init sends json without any content-type header
		// so we include "" in our case and parse the body as json
		var err error
		b, err = readClose(req.Body)
		if err != nil {
			j.Error(errors.WithMessage(err, "reading phone-home body"))
			w.WriteHeader(http.StatusBadRequest)

			return
		}
	case "application/x-www-form-urlencoded":
		// We convert urlencoded values to JSON
		if err := req.ParseForm(); err != nil {
			j.Error(errors.Wrap(err, "parsing http form"))
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		post_data := map[string]string{}
		for key, value := range req.PostForm {
			post_data[key] = value[0]
		}

		buf := new(bytes.Buffer)
		json.NewEncoder(buf).Encode(post_data)
		b = buf.Bytes()
	default:
		// Any other content types equal a bad request
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	j.phoneHome(req.Context(), b)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

func (j Job) ServeProblemEndpoint(w http.ResponseWriter, req *http.Request) {
	b, err := readClose(req.Body)
	if err != nil {
		j.Error(errors.WithMessage(err, "reading problem body"))
		w.WriteHeader(http.StatusBadRequest)

		return
	}
	var v struct {
		Problem string `json:"problem"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		j.Error(errors.Wrap(err, "parsing problem body as json"))
		w.WriteHeader(http.StatusBadRequest)

		return
	}
	if !j.PostHardwareProblem(req.Context(), v.Problem) {
		w.WriteHeader(http.StatusBadGateway)

		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

func readClose(r io.ReadCloser) (b []byte, err error) {
	b, err = ioutil.ReadAll(r)
	r.Close()

	return b, errors.Wrap(err, "reading file")
}
