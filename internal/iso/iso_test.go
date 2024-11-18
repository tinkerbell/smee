package iso

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"

	slogmulti "github.com/samber/slog-multi"
	slogsampling "github.com/samber/slog-sampling"
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
			// Will print 10% of entries.
			option := slogsampling.UniformSamplingOption{
				// The sample rate for sampling traces in the range [0.0, 1.0].
				Rate: 0.002,
			}

			logger := slog.New(
				slogmulti.
					Pipe(option.NewMiddleware()).
					Handler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})),
			)
			h := &Handler{
				parsedURL:    u,
				SampleLogger: logger,
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
