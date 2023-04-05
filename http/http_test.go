package http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestServeHTTP(t *testing.T) {
	c := &Config{
		GitRev:     "test rev",
		StartTime:  time.Now(),
		Logger:     logr.Discard(),
		IPXEScript: &IPXEScript{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go c.ServeHTTP(ctx, "127.0.0.1:31300", nil)

	time.Sleep(time.Microsecond * 500)
	res, err := http.Get("http://127.0.0.1:31300/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("not ok")
	}
	got, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"git_rev":"test rev"`) {
		t.Fatalf("healthz response does not contain matching 'git_rev', got: %v", string(got))
	}
}
