package main

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/backend"
)

/*

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
	osieURL:            "https://tinkerbell.org/hook/",
*/

func TestParser(t *testing.T) {
	want := config{
		syslog: syslogConfig{
			enabled:  true,
			bindAddr: "192.168.2.4:514",
		},
		tftp: tftp{
			enabled:  true,
			timeout:  5 * time.Second,
			bindAddr: "192.168.2.4:69",
		},
		ipxeHTTPBinary: ipxeHTTPBinary{
			enabled: true,
		},
		ipxeHTTPScript: ipxeHTTPScript{
			enabled:  true,
			bindAddr: "192.168.2.4:80",
		},
		dhcp: dhcpConfig{
			enabled:           true,
			bindAddr:          "0.0.0.0:67",
			ipForPacket:       "192.168.2.4",
			syslogIP:          "192.168.2.4",
			tftpIP:            "192.168.2.4:69",
			httpIpxeBinaryIP:  "http://192.168.2.4:8080/ipxe/",
			httpIpxeScriptURL: "http://192.168.2.4/auto.ipxe",
		},
		logLevel: "info",
		backends: dhcpBackends{
			file:       backend.File{},
			kubernetes: backend.Kube{Enabled: true},
		},
	}
	got := config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	args := []string{
		"-log-level", "info",
		"-syslog-addr", "192.168.2.4:514",
		"-tftp-addr", "192.168.2.4:69",
		"-http-addr", "192.168.2.4:80",
		"-dhcp-ip-for-packet", "192.168.2.4",
		"-dhcp-syslog-ip", "192.168.2.4",
		"-dhcp-tftp-ip", "192.168.2.4:69",
		"-dhcp-http-ipxe-binary-ip", "http://192.168.2.4:8080/ipxe/",
		"-dhcp-http-ipxe-script-url", "http://192.168.2.4/auto.ipxe",
	}
	cli := newCLI(&got, fs)
	cli.Parse(args)
	opts := cmp.Options{
		cmp.AllowUnexported(config{}),
		cmp.AllowUnexported(syslogConfig{}),
		cmp.AllowUnexported(tftp{}),
		cmp.AllowUnexported(ipxeHTTPBinary{}),
		cmp.AllowUnexported(ipxeHTTPScript{}),
		cmp.AllowUnexported(dhcpConfig{}),
		cmp.AllowUnexported(dhcpBackends{}),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Fatal(diff)
	}
}

func TestCustomUsageFunc(t *testing.T) {
	defaultIP := detectPublicIPv4("")
	want := fmt.Sprintf(`USAGE
  Run Boots server for provisioning

FLAGS
  -backend-file               [backend] enable the DHCP file backend (default "false")
  -backend-file-path          [backend] the hardware yaml file path for the file backend
  -backend-kube               [backend] enable the kubernetes backend (default "true")
  -backend-kube-api           [backend] the Kubernetes API URL, used for in-cluster client construction. Only applies if the kube backend is enabled
  -backend-kube-namespace     [backend] an optional Kubernetes namespace override to query hardware data from.
  -backend-kubeconfig         [backend] the Kubernetes config file location. Only applies if the kube backend is enabled
  -dhcp                       [dhcp] enable DHCP server(receiver) (default "true")
  -dhcp-addr                  [dhcp] local IP and port to listen on for DHCP requests (default "0.0.0.0:67")
  -dhcp-http-ipxe-binary-ip   [dhcp] http ipxe binary server IP address to use in DHCP packets (default "http://%[1]v:8080/ipxe/")
  -dhcp-http-ipxe-script-url  [dhcp] http ipxe script server URL to use in DHCP packets (default "http://%[1]v/auto.ipxe")
  -dhcp-ip-for-packet         [dhcp] ip address to use in DHCP packets (opt 54, etc) (default "%[1]v")
  -dhcp-syslog-ip             [dhcp] syslog server IP address to use in DHCP packets (opt 7) (default "%[1]v")
  -dhcp-tftp-ip               [dhcp] tftp server IP address to use in DHCP packets (opt 66, etc) (default "%[1]v:69")
  -extra-kernel-args          [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script.
  -http-addr                  [http] local IP and port to listen on for iPXE http script requests (default "%[1]v:80")
  -http-ipxe-binary           [http] enable iPXE http binary server(receiver) (default "true")
  -http-ipxe-script           [http] enable iPXE http script server(receiver) (default "true")
  -ipxe-script-patch          [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -log-level                  log level (debug, info) (default "info")
  -osie-url                   [http] url where OSIE(Hook) images are located.
  -syslog                     [syslog] enable syslog server(receiver) (default "true")
  -syslog-addr                [syslog] local IP and port to listen on for syslog messages (default "%[1]v:514")
  -tftp                       [tftp] enable iPXE tftp binary server(receiver) (default "true")
  -tftp-addr                  [tftp] local IP and port to listen on for iPXE tftp binary requests (default "%[1]v:69")
  -tftp-timeout               [tftp] iPXE tftp binary server requests timeout (default "5s")
  -tink-server                [http] ip:port for the Tink server.
  -tink-server-tls            [http] use TLS for Tink server. (default "false")
  -trusted-proxies            [http] comma separated list of trusted proxies
`, defaultIP)

	c := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cli := newCLI(c, fs)
	got := customUsageFunc(cli)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}
