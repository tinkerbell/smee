package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// customUsageFunc is a custom UsageFunc used for all commands.
func customUsageFunc(c *ffcli.Command) string {
	var b strings.Builder

	if c.LongHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", c.LongHelp)
	}

	fmt.Fprintf(&b, "USAGE\n")
	if c.ShortUsage != "" {
		fmt.Fprintf(&b, "  %s\n", c.ShortUsage)
	} else {
		fmt.Fprintf(&b, "  %s\n", c.Name)
	}
	fmt.Fprintf(&b, "\n")

	if len(c.Subcommands) > 0 {
		fmt.Fprintf(&b, "SUBCOMMANDS\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		for _, subcommand := range c.Subcommands {
			fmt.Fprintf(tw, "  %s\t%s\n", subcommand.Name, subcommand.ShortHelp)
		}
		tw.Flush()
		fmt.Fprintf(&b, "\n")
	}

	if countFlags(c.FlagSet) > 0 {
		fmt.Fprintf(&b, "FLAGS\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		type flagUsage struct {
			name         string
			usage        string
			defaultValue string
		}
		flags := []flagUsage{}
		c.FlagSet.VisitAll(func(f *flag.Flag) {
			f1 := flagUsage{name: f.Name, usage: f.Usage, defaultValue: f.DefValue}
			flags = append(flags, f1)
		})

		sort.SliceStable(flags, func(i, j int) bool {
			// sort by the service name between the brackets "[]" found in the usage string.
			r := regexp.MustCompile(`^\[(.*?)\]`)
			return r.FindString(flags[i].usage) < r.FindString(flags[j].usage)
		})
		for _, elem := range flags {
			if elem.defaultValue != "" {
				fmt.Fprintf(tw, "  -%s\t%s (default %q)\n", elem.name, elem.usage, elem.defaultValue)
			} else {
				fmt.Fprintf(tw, "  -%s\t%s\n", elem.name, elem.usage)
			}
		}
		tw.Flush()
		fmt.Fprintf(&b, "\n")
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func countFlags(fs *flag.FlagSet) (n int) {
	fs.VisitAll(func(*flag.Flag) { n++ })

	return n
}

func syslogFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.syslog.enabled, "syslog-enabled", true, "[syslog] enable Syslog server(receiver)")
	fs.StringVar(&c.syslog.bindAddr, "syslog-addr", detectPublicIPv4(), "[syslog] local IP to listen on for Syslog messages")
	fs.IntVar(&c.syslog.bindPort, "syslog-port", 514, "[syslog] local port to listen on for Syslog messages")
}

func tftpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.tftp.enabled, "tftp-enabled", true, "[tftp] enable iPXE TFTP binary server)")
	fs.StringVar(&c.tftp.bindAddr, "tftp-addr", detectPublicIPv4(), "[tftp] local IP to listen on for iPXE TFTP binary requests")
	fs.IntVar(&c.tftp.bindPort, "tftp-port", 69, "[tftp] local port to listen on for iPXE TFTP binary requests")
	fs.DurationVar(&c.tftp.timeout, "tftp-timeout", time.Second*5, "[tftp] iPXE TFTP binary server requests timeout")
	fs.StringVar(&c.tftp.ipxeScriptPatch, "ipxe-script-patch", "", "[tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP")
	fs.IntVar(&c.tftp.blockSize, "tftp-block-size", 512, "[tftp] TFTP block size a value between 512 (the default block size for TFTP) and 65456 (the max size a UDP packet payload can be)")
}

func ipxeHTTPBinaryFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPBinary.enabled, "http-ipxe-binary-enabled", true, "[http] enable iPXE HTTP binary server")
}

func ipxeHTTPScriptFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPScript.enabled, "http-ipxe-script-enabled", true, "[http] enable iPXE HTTP script server")
	fs.StringVar(&c.ipxeHTTPScript.bindAddr, "http-addr", detectPublicIPv4(), "[http] local IP to listen on for iPXE HTTP script requests")
	fs.IntVar(&c.ipxeHTTPScript.bindPort, "http-port", 8080, "[http] local port to listen on for iPXE HTTP script requests")
	fs.StringVar(&c.ipxeHTTPScript.extraKernelArgs, "extra-kernel-args", "", "[http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script")
	fs.StringVar(&c.ipxeHTTPScript.trustedProxies, "trusted-proxies", "", "[http] comma separated list of trusted proxies in CIDR notation")
	fs.StringVar(&c.ipxeHTTPScript.hookURL, "osie-url", "", "[http] URL where OSIE (HookOS) images are located")
	fs.StringVar(&c.ipxeHTTPScript.tinkServer, "tink-server", "", "[http] IP:Port for the Tink server")
	fs.BoolVar(&c.ipxeHTTPScript.tinkServerUseTLS, "tink-server-tls", false, "[http] use TLS for Tink server")
	fs.IntVar(&c.ipxeHTTPScript.retries, "ipxe-script-retries", 0, "[http] number of retries to attempt when fetching kernel and initrd files in the iPXE script")
	fs.IntVar(&c.ipxeHTTPScript.retryDelay, "ipxe-script-retry-delay", 2, "[http] delay (in seconds) between retries when fetching kernel and initrd files in the iPXE script")
}

func dhcpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.dhcp.enabled, "dhcp-enabled", true, "[dhcp] enable DHCP server")
	fs.StringVar(&c.dhcp.mode, "dhcp-mode", "reservation", "[dhcp] DHCP mode (reservation, proxy, auto-proxy)")
	fs.StringVar(&c.dhcp.bindAddr, "dhcp-addr", "0.0.0.0:67", "[dhcp] local IP:Port to listen on for DHCP requests")
	fs.StringVar(&c.dhcp.bindInterface, "dhcp-iface", "", "[dhcp] interface to bind to for DHCP requests")
	fs.StringVar(&c.dhcp.ipForPacket, "dhcp-ip-for-packet", detectPublicIPv4(), "[dhcp] IP address to use in DHCP packets (opt 54, etc)")
	fs.StringVar(&c.dhcp.syslogIP, "dhcp-syslog-ip", detectPublicIPv4(), "[dhcp] Syslog server IP address to use in DHCP packets (opt 7)")
	fs.StringVar(&c.dhcp.tftpIP, "dhcp-tftp-ip", detectPublicIPv4(), "[dhcp] TFTP server IP address to use in DHCP packets (opt 66, etc)")
	fs.IntVar(&c.dhcp.tftpPort, "dhcp-tftp-port", 69, "[dhcp] TFTP server port to use in DHCP packets (opt 66, etc)")
	fs.StringVar(&c.dhcp.httpIpxeBinaryURL.Scheme, "dhcp-http-ipxe-binary-scheme", "http", "[dhcp] HTTP iPXE binaries scheme to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeBinaryURL.Host, "dhcp-http-ipxe-binary-host", detectPublicIPv4(), "[dhcp] HTTP iPXE binaries host or IP to use in DHCP packets")
	fs.IntVar(&c.dhcp.httpIpxeBinaryURL.Port, "dhcp-http-ipxe-binary-port", 8080, "[dhcp] HTTP iPXE binaries port to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeBinaryURL.Path, "dhcp-http-ipxe-binary-path", "/ipxe/", "[dhcp] HTTP iPXE binaries path to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeScript.Scheme, "dhcp-http-ipxe-script-scheme", "http", "[dhcp] HTTP iPXE script scheme to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeScript.Host, "dhcp-http-ipxe-script-host", detectPublicIPv4(), "[dhcp] HTTP iPXE script host or IP to use in DHCP packets")
	fs.IntVar(&c.dhcp.httpIpxeScript.Port, "dhcp-http-ipxe-script-port", 8080, "[dhcp] HTTP iPXE script port to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeScript.Path, "dhcp-http-ipxe-script-path", "/auto.ipxe", "[dhcp] HTTP iPXE script path to use in DHCP packets")
	fs.BoolVar(&c.dhcp.httpIpxeScript.injectMacAddress, "dhcp-http-ipxe-script-prepend-mac", true, "[dhcp] prepend the hardware MAC address to iPXE script URL base, http://1.2.3.4/auto.ipxe -> http://1.2.3.4/40:15:ff:89:cc:0e/auto.ipxe")
}

func backendFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.backends.file.Enabled, "backend-file-enabled", false, "[backend] enable the file backend for DHCP and the HTTP iPXE script")
	fs.StringVar(&c.backends.file.FilePath, "backend-file-path", "", "[backend] the hardware yaml file path for the file backend")
	fs.BoolVar(&c.backends.kubernetes.Enabled, "backend-kube-enabled", true, "[backend] enable the kubernetes backend for DHCP and the HTTP iPXE script")
	fs.StringVar(&c.backends.kubernetes.ConfigFilePath, "backend-kube-config", "", "[backend] the Kubernetes config file location, kube backend only")
	fs.StringVar(&c.backends.kubernetes.APIURL, "backend-kube-api", "", "[backend] the Kubernetes API URL, used for in-cluster client construction, kube backend only")
	fs.StringVar(&c.backends.kubernetes.Namespace, "backend-kube-namespace", "", "[backend] an optional Kubernetes namespace override to query hardware data from, kube backend only")
	fs.BoolVar(&c.backends.Noop.Enabled, "backend-noop-enabled", false, "[backend] enable the noop backend for DHCP and the HTTP iPXE script")
}

func otelFlags(c *config, fs *flag.FlagSet) {
	fs.StringVar(&c.otel.endpoint, "otel-endpoint", "", "[otel] OpenTelemetry collector endpoint")
	fs.BoolVar(&c.otel.insecure, "otel-insecure", true, "[otel] OpenTelemetry collector insecure")
}

func setFlags(c *config, fs *flag.FlagSet) {
	fs.StringVar(&c.logLevel, "log-level", "info", "log level (debug, info)")
	dhcpFlags(c, fs)
	tftpFlags(c, fs)
	ipxeHTTPBinaryFlags(c, fs)
	ipxeHTTPScriptFlags(c, fs)
	syslogFlags(c, fs)
	backendFlags(c, fs)
	otelFlags(c, fs)
}

func newCLI(cfg *config, fs *flag.FlagSet) *ffcli.Command {
	setFlags(cfg, fs)
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "smee [flags]",
		LongHelp:   "Smee is the DHCP and Network boot service for use in the Tinkerbell stack.",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		UsageFunc:  customUsageFunc,
	}
}

func detectPublicIPv4() string {
	ip, err := autoDetectPublicIPv4()
	if err != nil {
		return ""
	}

	return ip.String()
}

func autoDetectPublicIPv4() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("unable to auto-detect public IPv4: %w", err)
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

		return v4, nil
	}

	return nil, errors.New("unable to auto-detect public IPv4")
}
