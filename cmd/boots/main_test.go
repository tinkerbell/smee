package main

import (
	"flag"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/ipxedust"
)

func TestParser(t *testing.T) {
	want := &config{
		ipxe: ipxedust.Command{
			TFTPAddr:             "0.0.0.0:69",
			TFTPTimeout:          time.Second * 5,
			EnableTFTPSinglePort: false,
		},
		ipxeTFTPEnabled:    true,
		ipxeHTTPEnabled:    true,
		ipxeRemoteTFTPAddr: "192.168.2.225",
		ipxeRemoteHTTPAddr: "192.168.2.225:8080",
		httpAddr:           "192.168.2.225:8080",
		dhcpAddr:           "0.0.0.0:67",
		syslogAddr:         "0.0.0.0:514",
		logLevel:           "info",
	}
	got := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	args := []string{
		"-log-level", "info",
		"-ipxe-remote-tftp-addr", "192.168.2.225",
		"-ipxe-remote-http-addr", "192.168.2.225:8080",
		"-http-addr", "192.168.2.225:8080",
		"-dhcp-addr", "0.0.0.0:67",
		"-syslog-addr", "0.0.0.0:514",
	}
	cli := newCLI(got, fs)
	cli.Parse(args)
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(ipxedust.Command{}, "Log"), cmp.AllowUnexported(config{})); diff != "" {
		t.Fatal(diff)
	}
}

func TestCustomUsageFunc(t *testing.T) {
	var defaultIP net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ip.IP.To4()
		if v4 == nil || !v4.IsGlobalUnicast() {
			continue
		}
		defaultIP = v4

		break
	}

	want := fmt.Sprintf(`USAGE
  Run Boots server for provisioning

FLAGS
  -dhcp-addr              IP and port to listen on for DHCP. (default "%v:67")
  -http-addr              local IP and port to listen on for the serving iPXE binaries and files via HTTP. (default "%[1]v:80")
  -ipxe-enable-http       enable serving iPXE binaries via HTTP. (default "true")
  -ipxe-enable-tftp       enable serving iPXE binaries via TFTP. (default "true")
  -ipxe-remote-http-addr  remote IP and port where iPXE binaries are served via HTTP. Overrides -http-addr for iPXE binaries only.
  -ipxe-remote-tftp-addr  remote IP where iPXE binaries are served via TFTP. Overrides -tftp-addr.
  -ipxe-tftp-addr         local IP and port to listen on for serving iPXE binaries via TFTP (port must be 69). (default "0.0.0.0:69")
  -ipxe-tftp-timeout      local iPXE TFTP server requests timeout. (default "5s")
  -log-level              log level. (default "info")
  -syslog-addr            IP and port to listen on for syslog messages. (default "%[1]v:514")
`, defaultIP)
	c := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cli := newCLI(c, fs)
	got := customUsageFunc(cli)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatal(diff)
	}
}
