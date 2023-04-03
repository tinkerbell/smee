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

func newCLI(cfg *config, fs *flag.FlagSet) *ffcli.Command {
	fs.StringVar(&cfg.ipxe.TFTPAddr, "ipxe-tftp-addr", "0.0.0.0:69", "local IP and port to listen on for serving iPXE binaries via TFTP (port must be 69).")
	fs.DurationVar(&cfg.ipxe.TFTPTimeout, "ipxe-tftp-timeout", time.Second*5, "local iPXE TFTP server requests timeout.")
	fs.BoolVar(&cfg.ipxeTFTPEnabled, "ipxe-enable-tftp", true, "enable serving iPXE binaries via TFTP.")
	fs.BoolVar(&cfg.ipxeHTTPEnabled, "ipxe-enable-http", true, "enable serving iPXE binaries via HTTP.")
	fs.StringVar(&cfg.ipxeRemoteTFTPAddr, "ipxe-remote-tftp-addr", "", "remote IP where iPXE binaries are served via TFTP. Overrides -tftp-addr.")
	fs.StringVar(&cfg.ipxeRemoteHTTPAddr, "ipxe-remote-http-addr", "", "remote IP and port where iPXE binaries are served via HTTP. Overrides -http-addr for iPXE binaries only.")
	fs.StringVar(&cfg.ipxeVars, "ipxe-vars", "", "additional variable definitions to include in all iPXE installer scripts. Separate multiple var definitions with spaces, e.g. 'var1=val1 var2=val2'.")
	fs.StringVar(&cfg.httpAddr, "http-addr", httpBind, "local IP and port to listen on for the serving iPXE binaries and files via HTTP.")
	fs.StringVar(&cfg.logLevel, "log-level", "info", "log level.")
	fs.StringVar(&cfg.dhcpAddr, "dhcp-addr", bootpBind, "IP and port to listen on for DHCP.")
	fs.StringVar(&cfg.syslogAddr, "syslog-addr", syslogBind, "IP and port to listen on for syslog messages.")
	fs.StringVar(&cfg.extraKernelArgs, "extra-kernel-args", "", "Extra set of kernel args (k=v k=v) that are appended to the kernel cmdline when booting via iPXE.")
	fs.StringVar(&cfg.kubeconfig, "kubeconfig", "", "The Kubernetes config file location. Only applies if DATA_MODEL_VERSION=kubernetes.")
	fs.StringVar(&cfg.kubeAPI, "kubernetes", "", "The Kubernetes API URL, used for in-cluster client construction. Only applies if DATA_MODEL_VERSION=kubernetes.")
	fs.StringVar(&cfg.kubeNamespace, "kube-namespace", "", "An optional Kubernetes namespace override to query hardware data from.")
	fs.StringVar(&cfg.osieURL, "osie-url", "", "URL where OSIE/Hook images are located.")
	fs.StringVar(&cfg.ipxeScriptPatch, "ipxe-script-patch", "", "iPXE script fragment to patch into served iPXE binaries served via TFTP and HTTP")

	return &ffcli.Command{
		Name:       name,
		ShortUsage: "Run Boots server for provisioning",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		UsageFunc:  customUsageFunc,
	}
}
