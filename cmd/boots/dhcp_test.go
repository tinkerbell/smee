package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/google/go-cmp/cmp"
	"github.com/packethost/dhcp4-go"
	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
)

func TestGetCircuitID(t *testing.T) {

	for _, test := range []struct {
		name        string
		option      dhcp4.Option
		optionvalue []byte
		expected    string
		err         string // logged error description
	}{
		{
			name:        "With option82 circuitid",
			option:      dhcp4.OptionRelayAgentInformation,
			optionvalue: []byte("\x01\x19esr1.d11.lab1:ge-1/0/47.0\x02\x0Bge-1/0/47.0"),
			expected:    "esr1.d11.lab1:ge-1/0/47",
			err:         "",
		},
		{
			name:        "No option82 information",
			option:      dhcp4.OptionEnd, // option not important here, just needs to not have OptionRelayAgentInformation
			optionvalue: []byte{},
			expected:    "",
			err:         "option82 information not available for this mac",
		},
		{
			name:        "Malformed option82",
			option:      dhcp4.OptionRelayAgentInformation,
			optionvalue: []byte("\x01\x19esr1.d11.la"),
			expected:    "",
			err:         "option82 option1 out of bounds (check eightytwo[1])",
		},
	} {
		t.Log(test.name)
		var packet = new(dhcp4.Packet)

		packet.OptionMap = make(dhcp4.OptionMap, 255)
		packet.SetOption(test.option, test.optionvalue)
		c, err := getCircuitID(packet)
		if err != nil {
			if err.Error() != test.err {
				t.Fatalf("unexpected error, want: %s, got: %s", test.err, err)
			}
		}
		if c != "" {
			if c != test.expected {
				t.Fatalf("expected value not returned for option82, want: %s, got: %s", test.expected, c)
			}
		}

	}
}

func TestMain(m *testing.M) {
	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")
	metrics.Init(l)
	os.Exit(m.Run())
}

func TestServeJobFile(t *testing.T) {
	tests := map[string]struct {
		expectedResp       []byte
		expectedStatusCode int
		allowPxe           bool
		err                error
	}{
		"success":                   {expectedResp: []byte("success"), expectedStatusCode: http.StatusOK, allowPxe: true},
		"fail createFromRemoteAddr": {expectedStatusCode: http.StatusNotFound, err: fmt.Errorf("failed")},
		"fail allowPxe is false":    {expectedStatusCode: http.StatusNotFound},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			monkey.Patch(job.CreateFromRemoteAddr, func(_ context.Context, _ string) (job.Job, error) {
				return job.Job{}, tc.err
			})
			var jb job.Job
			monkey.PatchInstanceMethod(reflect.TypeOf(jb), "AllowPxe", func(_ job.Job) bool {
				return tc.allowPxe
			})
			monkey.PatchInstanceMethod(reflect.TypeOf(jb), "ServeFile", func(_ job.Job, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.expectedStatusCode)
				_, _ = w.Write(tc.expectedResp)
			})

			w := httptest.NewRecorder()
			req := http.Request{RemoteAddr: "127.0.0.1:80"}
			serveJobFile(w, &req)
			if diff := cmp.Diff(tc.expectedResp, w.Body.Bytes()); diff != "" {
				t.Fatal(diff)
			}
			if tc.expectedStatusCode != w.Result().StatusCode {
				t.Fatalf("expected: %v, got: %v", tc.expectedStatusCode, w.Result().StatusCode)
			}
		})
	}
}
