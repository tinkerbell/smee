package main

import (
	"flag"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/ipxedust"
)

func TestParser(t *testing.T) {
	want := &config{
		ipxe: ipxedust.Command{
			TFTPAddr:             "0.0.0.0",
			TFTPTimeout:          time.Second * 5,
			HTTPAddr:             "0.0.0.0:8080",
			HTTPTimeout:          time.Second * 5,
			EnableTFTPSinglePort: false,
		},
		iTFTPDisabled:   false,
		iHTTPDisabled:   false,
		remoteTFTPAddr:  "192.168.2.225",
		remoteiHTTPAddr: "192.168.2.225:8080",
		httpAddr:        "192.168.2.225:8080",
		dhcpAddr:        "0.0.0.0:67",
		syslogAddr:      "0.0.0.0:514",
		logLevel:        "info",
	}
	got := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	args := []string{
		"-log-level", "info",
		"-remote-tftp-addr", "192.168.2.225",
		"-remote-ihttp-addr", "192.168.2.225:8080",
		"-http-addr", "192.168.2.225:8080",
		"-dhcp-addr", "0.0.0.0:67",
		"-syslog-addr", "0.0.0.0:514",
	}
	parser(got, fs, args)
	if diff := cmp.Diff(got, want, cmpopts.IgnoreFields(ipxedust.Command{}, "Log"), cmp.AllowUnexported(config{})); diff != "" {
		t.Fatal(diff)
	}
}
