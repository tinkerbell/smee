package main

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParser(t *testing.T) {
	want := config{
		syslog: syslogConfig{
			enabled:  true,
			bindAddr: "192.168.2.4",
			bindPort: 514,
		},
		tftp: tftp{
			blockSize: 512,
			enabled:   true,
			timeout:   5 * time.Second,
			bindAddr:  "192.168.2.4",
			bindPort:  69,
		},
		ipxeHTTPBinary: ipxeHTTPBinary{
			enabled: true,
		},
		ipxeHTTPScript: ipxeHTTPScript{
			enabled:    true,
			bindAddr:   "192.168.2.4",
			bindPort:   8080,
			retryDelay: 2,
		},
		dhcp: dhcpConfig{
			enabled:     true,
			mode:        "reservation",
			bindAddr:    "0.0.0.0:67",
			ipForPacket: "192.168.2.4",
			syslogIP:    "192.168.2.4",
			tftpIP:      "192.168.2.4",
			tftpPort:    69,
			httpIpxeBinaryURL: urlBuilder{
				Scheme: "http",
				Host:   "192.168.2.4",
				Port:   8080,
				Path:   "/ipxe/",
			},
			httpIpxeScript: httpIpxeScript{
				urlBuilder: urlBuilder{
					Scheme: "http",
					Host:   "192.168.2.4",
					Port:   8080,
					Path:   "/auto.ipxe",
				},
				injectMacAddress: true,
			},
		},
		logLevel: "info",
		backends: dhcpBackends{
			file:       File{},
			kubernetes: Kube{Enabled: true},
		},
		otel: otelConfig{
			insecure: true,
		},
	}
	got := config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	args := []string{
		"-log-level", "info",
		"-syslog-addr", "192.168.2.4",
		"-tftp-addr", "192.168.2.4",
		"-http-addr", "192.168.2.4",
		"-dhcp-ip-for-packet", "192.168.2.4",
		"-dhcp-syslog-ip", "192.168.2.4",
		"-dhcp-tftp-ip", "192.168.2.4",
		"-dhcp-http-ipxe-binary-host", "192.168.2.4",
		"-dhcp-http-ipxe-script-host", "192.168.2.4",
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
		cmp.AllowUnexported(httpIpxeScript{}),
		cmp.AllowUnexported(otelConfig{}),
		cmp.AllowUnexported(urlBuilder{}),
	}
	if diff := cmp.Diff(want, got, opts); diff != "" {
		t.Fatal(diff)
	}
}

func TestCustomUsageFunc(t *testing.T) {
	defaultIP := detectPublicIPv4()
	want := fmt.Sprintf(`Smee is the DHCP and Network boot service for use in the Tinkerbell stack.

USAGE
  smee [flags]

FLAGS
  -log-level                          log level (debug, info) (default "info")
  -backend-file-enabled               [backend] enable the file backend for DHCP and the HTTP iPXE script (default "false")
  -backend-file-path                  [backend] the hardware yaml file path for the file backend
  -backend-kube-api                   [backend] the Kubernetes API URL, used for in-cluster client construction, kube backend only
  -backend-kube-config                [backend] the Kubernetes config file location, kube backend only
  -backend-kube-enabled               [backend] enable the kubernetes backend for DHCP and the HTTP iPXE script (default "true")
  -backend-kube-namespace             [backend] an optional Kubernetes namespace override to query hardware data from, kube backend only
  -backend-noop-enabled               [backend] enable the noop backend for DHCP and the HTTP iPXE script (default "false")
  -dhcp-addr                          [dhcp] local IP:Port to listen on for DHCP requests (default "0.0.0.0:67")
  -dhcp-enabled                       [dhcp] enable DHCP server (default "true")
  -dhcp-http-ipxe-binary-host         [dhcp] HTTP iPXE binaries host or IP to use in DHCP packets (default "%[1]v")
  -dhcp-http-ipxe-binary-path         [dhcp] HTTP iPXE binaries path to use in DHCP packets (default "/ipxe/")
  -dhcp-http-ipxe-binary-port         [dhcp] HTTP iPXE binaries port to use in DHCP packets (default "8080")
  -dhcp-http-ipxe-binary-scheme       [dhcp] HTTP iPXE binaries scheme to use in DHCP packets (default "http")
  -dhcp-http-ipxe-script-host         [dhcp] HTTP iPXE script host or IP to use in DHCP packets (default "%[1]v")
  -dhcp-http-ipxe-script-path         [dhcp] HTTP iPXE script path to use in DHCP packets (default "/auto.ipxe")
  -dhcp-http-ipxe-script-port         [dhcp] HTTP iPXE script port to use in DHCP packets (default "8080")
  -dhcp-http-ipxe-script-prepend-mac  [dhcp] prepend the hardware MAC address to iPXE script URL base, http://1.2.3.4/auto.ipxe -> http://1.2.3.4/40:15:ff:89:cc:0e/auto.ipxe (default "true")
  -dhcp-http-ipxe-script-scheme       [dhcp] HTTP iPXE script scheme to use in DHCP packets (default "http")
  -dhcp-http-ipxe-script-url          [dhcp] HTTP iPXE script URL to use in DHCP packets, this overrides the flags for dhcp-http-ipxe-script-{scheme, host, port, path}
  -dhcp-iface                         [dhcp] interface to bind to for DHCP requests
  -dhcp-ip-for-packet                 [dhcp] IP address to use in DHCP packets (opt 54, etc) (default "%[1]v")
  -dhcp-mode                          [dhcp] DHCP mode (reservation, proxy, auto-proxy) (default "reservation")
  -dhcp-syslog-ip                     [dhcp] Syslog server IP address to use in DHCP packets (opt 7) (default "%[1]v")
  -dhcp-tftp-ip                       [dhcp] TFTP server IP address to use in DHCP packets (opt 66, etc) (default "%[1]v")
  -dhcp-tftp-port                     [dhcp] TFTP server port to use in DHCP packets (opt 66, etc) (default "69")
  -extra-kernel-args                  [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script
  -http-addr                          [http] local IP to listen on for iPXE HTTP script requests (default "%[1]v")
  -http-ipxe-binary-enabled           [http] enable iPXE HTTP binary server (default "true")
  -http-ipxe-script-enabled           [http] enable iPXE HTTP script server (default "true")
  -http-port                          [http] local port to listen on for iPXE HTTP script requests (default "8080")
  -ipxe-script-retries                [http] number of retries to attempt when fetching kernel and initrd files in the iPXE script (default "0")
  -ipxe-script-retry-delay            [http] delay (in seconds) between retries when fetching kernel and initrd files in the iPXE script (default "2")
  -osie-url                           [http] URL where OSIE (HookOS) images are located
  -tink-server                        [http] IP:Port for the Tink server
  -tink-server-tls                    [http] use TLS for Tink server (default "false")
  -trusted-proxies                    [http] comma separated list of trusted proxies in CIDR notation
  -otel-endpoint                      [otel] OpenTelemetry collector endpoint
  -otel-insecure                      [otel] OpenTelemetry collector insecure (default "true")
  -syslog-addr                        [syslog] local IP to listen on for Syslog messages (default "%[1]v")
  -syslog-enabled                     [syslog] enable Syslog server(receiver) (default "true")
  -syslog-port                        [syslog] local port to listen on for Syslog messages (default "514")
  -ipxe-script-patch                  [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -tftp-addr                          [tftp] local IP to listen on for iPXE TFTP binary requests (default "%[1]v")
  -tftp-block-size                    [tftp] TFTP block size a value between 512 (the default block size for TFTP) and 65456 (the max size a UDP packet payload can be) (default "512")
  -tftp-enabled                       [tftp] enable iPXE TFTP binary server) (default "true")
  -tftp-port                          [tftp] local port to listen on for iPXE TFTP binary requests (default "69")
  -tftp-timeout                       [tftp] iPXE TFTP binary server requests timeout (default "5s")
`, defaultIP)

	c := &config{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cli := newCLI(c, fs)
	got := customUsageFunc(cli)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}
