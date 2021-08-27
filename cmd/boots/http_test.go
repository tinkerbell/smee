package main

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type tclient struct {
	id      string
	getErr  error
	postErr error
}

func (c tclient) GetInstanceIDFromIP(ctx context.Context, ip net.IP) (string, error) {
	return c.id, c.getErr
}

func (c tclient) PostInstanceEvent(context.Context, string, io.Reader) (string, error) {
	return "", c.postErr
}

func TestServeEvents(t *testing.T) {
	for _, test := range []struct {
		name    string
		remote  string
		id      string
		getErr  error
		postErr error
		code    int
		body    string
		err     string // description of error logged due to scenario
	}{
		{name: "no remote",
			remote: "",
			err:    `split host port: missing port in address`,
		},
		{name: "no remote ip",
			remote: "localhost",
			err:    `split host port: address localhost: missing port in address`,
		},
		{name: "fake error in GetDeviceIDFromIP",
			remote: "10.0.0.1:42", code: 200, getErr: errors.New("fake error from GetDeviceIDFromIP"),
			err: `client=10.0.0.1:42 msg="no device found for client address"`,
		},
		{name: "no GetDeviceIDFromIP error, yet no id",
			remote: "10.0.0.1:42", code: 200,
			err: `client=10.0.0.1:42 msg="no device found for client address"`,
		},
		{name: "empty userEvent body",
			remote: "10.0.0.1:42", id: "id",
			err: "userEvent body is empty",
		},
		{name: "invalid userEvent json",
			remote: "10.0.0.1:42", id: "id", body: `invalid json`,
			err: "userEvent cannot be generated from supplied json",
		},
		{name: "failed to post userEvent",
			remote: "10.0.0.1:42", id: "id", body: `{}`, postErr: errors.New("fake PostInstanceUserEvent error"),
			err: "failed to post userEvent",
		},
		{name: "ok",
			remote: "10.0.0.1:42", id: "id", body: `{}`, code: 200,
			err: "",
		},
	} {
		t.Log(test.name)

		if test.code == 0 {
			test.code = http.StatusBadRequest
		}
		c := tclient{
			id:      test.id,
			getErr:  test.getErr,
			postErr: test.postErr,
		}

		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		req.RemoteAddr = test.remote
		req.Body = ioutil.NopCloser(strings.NewReader(test.body))
		w := httptest.NewRecorder()

		_, err := serveEvents(c, w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) != 0 {
			t.Fatal("expected empty body, got:", string(body))
		}

		if resp.StatusCode != test.code {
			t.Fatalf("unexpected response code, want: %d, got: %d", test.code, resp.StatusCode)
		}

		if resp.StatusCode == http.StatusOK {
			continue
		}

		if err.Error() != test.err {
			t.Fatalf("error mismatch, want: `%s`, got: `%s`", test.err, err.Error())
		}
	}
}
