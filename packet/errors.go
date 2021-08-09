package packet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

func IsNotExist(err error) bool {
	if e, ok := err.(*httpError); ok && e.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}

type httpError struct {
	StatusCode int
	Errors     []error
}

func (e *httpError) Error() string {
	switch len(e.Errors) {
	case 0:
		return fmt.Sprintf("HTTP %d", e.StatusCode)
	case 1:
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Errors[0])
	}
	errs := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		errs = append(errs, err.Error())
	}

	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, strings.Join(errs, "; "))
}

func (e *httpError) WrappedErrors() []error {
	return e.Errors
}

func (e *httpError) unmarshalErrors(r io.Reader) {
	var v struct {
		Error  string   `json:"error"`
		Errors []string `json:"errors"`
	}
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		e.Errors = []error{errors.Wrap(err, "unmarshalling errors body")}

		return
	}
	if n := len(v.Errors); n > 0 {
		errs := make([]error, len(v.Errors))
		for i := range v.Errors {
			errs[i] = errors.New(v.Errors[i])
		}
		e.Errors = errs
	} else if v.Error != "" {
		e.Errors = []error{errors.New(v.Error)}
	}
}
