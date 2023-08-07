package main

import (
	"flag"
	"fmt"
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

		sort.Slice(flags, func(i, j int) bool {
			// sort by the service name between the brackets "[]" found in the usage string.
			r := regexp.MustCompile(`^\[(.*?)\]`)
			iflag := r.FindString(flags[i].usage)
			jflag := r.FindString(flags[j].usage)
			return iflag < jflag
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
	fs.BoolVar(&c.syslog.enabled, "syslog-enabled", true, "[syslog] enable syslog server(receiver)")
	fs.StringVar(&c.syslog.bindAddr, "syslog-addr", detectPublicIPv4(":514"), "[syslog] local IP and port to listen on for syslog messages")
}

func tftpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.tftp.enabled, "tftp-enbled", true, "[tftp] enable iPXE tftp binary server)")
	fs.StringVar(&c.tftp.bindAddr, "tftp-addr", detectPublicIPv4(":69"), "[tftp] local IP and port to listen on for iPXE tftp binary requests")
	fs.DurationVar(&c.tftp.timeout, "tftp-timeout", time.Second*5, "[tftp] iPXE tftp binary server requests timeout")
	fs.StringVar(&c.tftp.ipxeScriptPatch, "ipxe-script-patch", "", "[tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP")
}

func ipxeHTTPBinaryFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPBinary.enabled, "http-ipxe-binary-enabled", true, "[http] enable iPXE http binary server")
}

func ipxeHTTPScriptFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.ipxeHTTPScript.enabled, "http-ipxe-script-enabled", true, "[http] enable iPXE http script server)")
	fs.StringVar(&c.ipxeHTTPScript.bindAddr, "http-addr", detectPublicIPv4(":80"), "[http] local IP and port to listen on for iPXE http script requests")
	fs.StringVar(&c.ipxeHTTPScript.extraKernelArgs, "extra-kernel-args", "", "[http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script")
	fs.StringVar(&c.ipxeHTTPScript.trustedProxies, "trusted-proxies", "", "[http] comma separated list of trusted proxies")
	fs.StringVar(&c.ipxeHTTPScript.hookURL, "osie-url", "", "[http] url where OSIE(Hook) images are located")
	fs.StringVar(&c.ipxeHTTPScript.tinkServer, "tink-server", "", "[http] ip:port for the Tink server")
	fs.BoolVar(&c.ipxeHTTPScript.tinkServerUseTLS, "tink-server-tls", false, "[http] use TLS for Tink server")
}

func dhcpFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.dhcp.enabled, "dhcp-enabled", true, "[dhcp] enable DHCP server")
	fs.StringVar(&c.dhcp.bindAddr, "dhcp-addr", "0.0.0.0:67", "[dhcp] local IP and port to listen on for DHCP requests")
	fs.StringVar(&c.dhcp.ipForPacket, "dhcp-ip-for-packet", detectPublicIPv4(""), "[dhcp] ip address to use in DHCP packets (opt 54, etc)")
	fs.StringVar(&c.dhcp.syslogIP, "dhcp-syslog-ip", detectPublicIPv4(""), "[dhcp] syslog server IP address to use in DHCP packets (opt 7)")
	fs.StringVar(&c.dhcp.tftpIP, "dhcp-tftp-ip", detectPublicIPv4(":69"), "[dhcp] tftp server IP address to use in DHCP packets (opt 66, etc)")
	fs.StringVar(&c.dhcp.httpIpxeBinaryIP, "dhcp-http-ipxe-binary-ip", "http://"+detectPublicIPv4(":8080/ipxe/"), "[dhcp] http ipxe binary server IP address to use in DHCP packets")
	fs.StringVar(&c.dhcp.httpIpxeScriptURL, "dhcp-http-ipxe-script-url", "http://"+detectPublicIPv4("/auto.ipxe"), "[dhcp] http ipxe script server URL to use in DHCP packets")
}

func backendFlags(c *config, fs *flag.FlagSet) {
	fs.BoolVar(&c.backends.file.Enabled, "backend-file-enabled", false, "[backend] enable the file backend for DHCP and the HTTP iPXE script")
	fs.StringVar(&c.backends.file.FilePath, "backend-file-path", "", "[backend] the hardware yaml file path for the file backend")
	fs.BoolVar(&c.backends.kubernetes.Enabled, "backend-kube-enabled", true, "[backend] enable the kubernetes backend for DHCP and the HTTP iPXE script")
	fs.StringVar(&c.backends.kubernetes.ConfigFilePath, "backend-kubeconfig", "", "[backend] the Kubernetes config file location, kube backend only")
	fs.StringVar(&c.backends.kubernetes.APIURL, "backend-kube-api", "", "[backend] the Kubernetes API URL, used for in-cluster client construction, kube backend only")
	fs.StringVar(&c.backends.kubernetes.Namespace, "backend-kube-namespace", "", "[backend] an optional Kubernetes namespace override to query hardware data from, kube backend only")
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
