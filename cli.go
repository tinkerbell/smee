package main

import (
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// customUsageFunc is a custom UsageFunc used for all commands.
func customUsageFunc(c *ffcli.Command) string {
	var b strings.Builder

	fmt.Fprintf(&b, "USAGE\n")
	if c.ShortUsage != "" {
		fmt.Fprintf(&b, "  %s\n", c.ShortUsage)
	} else {
		fmt.Fprintf(&b, "  %s\n", c.Name)
	}
	fmt.Fprintf(&b, "\n")

	if c.LongHelp != "" {
		fmt.Fprintf(&b, "%s\n\n", c.LongHelp)
	}

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
		c.FlagSet.VisitAll(func(f *flag.Flag) {
			format := "  -%s\t%s\n"
			values := []interface{}{f.Name, f.Usage}
			if def := f.DefValue; def != "" {
				format = "  -%s\t%s (default %q)\n"
				values = []interface{}{f.Name, f.Usage, def}
			}
			fmt.Fprintf(tw, format, values...)
		})
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
	fs.BoolVar(&c.syslog.enabled, "syslog", true, "[syslog] enable syslog server(receiver)")
	fs.StringVar(&c.syslog.bindAddr, "syslog-addr", detectPublicIPv4(":514"), "[syslog] local IP and port to listen on for syslog messages")
}

func tftpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.tftp.enabled, "tftp", true, "[tftp] enable iPXE tftp binary server(receiver)")
	fs.StringVar(&c.tftp.bindAddr, "tftp-addr", detectPublicIPv4(":69"), "[tftp] local IP and port to listen on for iPXE tftp binary requests")
	fs.DurationVar(&c.tftp.timeout, "tftp-timeout", time.Second*5, "[tftp] iPXE tftp binary server requests timeout")
	fs.StringVar(&c.tftp.ipxeScriptPatch, "ipxe-script-patch", "", "[tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP")
}

func ipxeHTTPBinaryFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPBinary.enabled, "http-ipxe-binary", true, "[http] enable iPXE http binary server(receiver)")
}

func ipxeHTTPScriptFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPScript.enabled, "http-ipxe-script", true, "[http] enable iPXE http script server(receiver)")
	fs.StringVar(&c.ipxeHTTPScript.bindAddr, "http-addr", detectPublicIPv4(":80"), "[http] local IP and port to listen on for iPXE http script requests")
	fs.StringVar(&c.ipxeHTTPScript.extraKernelArgs, "extra-kernel-args", "", "[http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script.")
	fs.StringVar(&c.ipxeHTTPScript.trustedProxies, "trusted-proxies", "", "[http] comma separated list of trusted proxies")
	fs.StringVar(&c.ipxeHTTPScript.hookURL, "osie-url", "", "[http] url where OSIE(Hook) images are located.")
	fs.StringVar(&c.ipxeHTTPScript.tinkServer, "tink-server", "", "[http] ip:port for the Tink server.")
	fs.BoolVar(&c.ipxeHTTPScript.tinkServerUseTLS, "tink-server-tls", false, "[http] use TLS for Tink server.")
}

func dhcpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.dhcp.enabled, "dhcp", true, "[dhcp] enable DHCP server(receiver)")
	fs.StringVar(&c.dhcp.bindAddr, "dhcp-addr", "0.0.0.0:67", "[dhcp] local IP and port to listen on for DHCP requests")
	fs.StringVar(&c.dhcp.ipForPacket, "dhcp-ip-for-packet", detectPublicIPv4(""), "[dhcp] ip address to use in DHCP packets (opt 54, etc)")
	fs.StringVar(&c.dhcp.syslogIP, "dhcp-syslog-ip", detectPublicIPv4(""), "[dhcp] syslog server IP address to use in DHCP packets (opt 7)")
	fs.StringVar(&c.dhcp.tftpIP, "dhcp-tftp-ip", detectPublicIPv4(":69"), "[dhcp] tftp server IP address to use in DHCP packets (opt 66, etc)")
	fs.StringVar(&c.dhcp.httpIpxeBinaryIP, "dhcp-http-ipxe-binary-ip", "http://"+detectPublicIPv4(":8080/ipxe/"), "[dhcp] http ipxe binary server IP address to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeScriptURL, "dhcp-http-ipxe-script-url", "http://"+detectPublicIPv4("/auto.ipxe"), "[dhcp] http ipxe script server URL to use in DHCP packets")
}

func backendFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.backends.file.Enabled, "backend-file", false, "[backend] enable the DHCP file backend")
	fs.StringVar(&c.backends.file.FilePath, "backend-file-path", "", "[backend] the hardware yaml file path for the file backend")
	fs.BoolVar(&c.backends.kubernetes.Enabled, "backend-kube", true, "[backend] enable the kubernetes backend")
	fs.StringVar(&c.backends.kubernetes.ConfigFilePath, "backend-kubeconfig", "", "[backend] the Kubernetes config file location. Only applies if the kube backend is enabled")
	fs.StringVar(&c.backends.kubernetes.APIURL, "backend-kube-api", "", "[backend] the Kubernetes API URL, used for in-cluster client construction. Only applies if the kube backend is enabled")
	fs.StringVar(&c.backends.kubernetes.Namespace, "backend-kube-namespace", "", "[backend] an optional Kubernetes namespace override to query hardware data from.")
}

func setFlags(c *config, fs *flag.FlagSet) {
	fs.StringVar(&c.logLevel, "log-level", "info", "log level (debug, info)")
	dhcpFlags(c, fs)
	tftpFlags(c, fs)
	ipxeHTTPBinaryFlags(c, fs)
	ipxeHTTPScriptFlags(c, fs)
	syslogFlags(c, fs)
	backendFlags(c, fs)
}

func newCLI(cfg *config, fs *flag.FlagSet) *ffcli.Command {
	setFlags(cfg, fs)
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "Run Boots server for provisioning",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		UsageFunc:  customUsageFunc,
	}
}
