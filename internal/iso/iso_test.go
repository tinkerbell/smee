package iso

import (
	"net/http"
	"net/url"
	"testing"
)

func TestReqPathInvalid(t *testing.T) {
	tests := map[string]struct {
		isoURL     string
		statusCode int
	}{
		"invalid URL prefix": {isoURL: "invalid", statusCode: http.StatusNotFound},
		"invalid URL":        {isoURL: "http://invalid.:123/hook.iso", statusCode: http.StatusBadRequest},
		"no script or url":   {isoURL: "http://10.10.10.10:8080/aa:aa:aa:aa:aa:aa/invalid.iso", statusCode: http.StatusInternalServerError},
	}
	for name, tt := range tests {
		u, _ := url.Parse(tt.isoURL)
		t.Run(name, func(t *testing.T) {
			h := &Handler{
				parsedURL: u,
			}
			req := http.Request{
				Method: http.MethodGet,
				URL:    u,
			}

			got, err := h.RoundTrip(&req)
			got.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if got.StatusCode != tt.statusCode {
				t.Fatalf("got response status code: %d, want status code: %d", got.StatusCode, tt.statusCode)
			}
		})
	}
}
