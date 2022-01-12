package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/avast/retry-go"
	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/packethost/pkg/env"
	"github.com/packethost/pkg/log"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/packet"
	"github.com/tinkerbell/boots/syslog"
	"github.com/tinkerbell/ipxedust"
	"github.com/tinkerbell/ipxedust/ihttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"inet.af/netaddr"

	"github.com/tinkerbell/boots/installers/coreos"
	"github.com/tinkerbell/boots/installers/custom_ipxe"
	"github.com/tinkerbell/boots/installers/nixos"
	"github.com/tinkerbell/boots/installers/osie"
	"github.com/tinkerbell/boots/installers/rancher"
	"github.com/tinkerbell/boots/installers/vmware"
)

var (
	client                packet.Client
	apiBaseURL            = env.URL("API_BASE_URL", "https://api.packet.net")
	provisionerEngineName = env.Get("PROVISIONER_ENGINE_NAME", "packet")

	mainlog log.Logger

	GitRev    = "unknown (use make)"
	StartTime = time.Now()
)

const name = "boots"

type config struct {
	// ipxe holds the config for serving ipxe binaries
	ipxe ipxedust.Command
	// iTFTPDisabled determines if local iPXE binaries served via TFTP are disabled
	iTFTPDisabled bool
	// iHTTPDisabled determines if local iPXE binaries served via TFTP are disabled
	iHTTPDisabled bool
	// remoteTFTPAddr is the address of the remote TFTP server serving iPXE binaries
	remoteTFTPAddr string
	// remoteTFTPPort is the address of the remote HTTP server serving iPXE binaries
	remoteiHTTPAddr string
	// httpAddr is the address of the HTTP server serving the iPXE script and other installer assets
	httpAddr string
	// dhcpAddr is the local address for the DHCP server
	dhcpAddr string
	// syslogAddr is the local address for the syslog server
	syslogAddr string
	// loglevel is the log level for boots
	logLevel string
}

func main() {
	cfg := &config{}
	cli := newCLI(cfg, flag.NewFlagSet(name, flag.ExitOnError))
	cli.Parse(os.Args[1:])

	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, name)
	defer otelShutdown(ctx)

	metrics.Init(l)
	dhcp.Init(l)
	conf.Init(l)
	httplog.Init(l)
	installers.Init(l)
	job.Init(l)
	syslog.Init(l)
	mainlog.With("version", GitRev).Info("starting")

	consumer := env.Get("API_CONSUMER_TOKEN")
	if consumer == "" {
		err := errors.New("required envvar missing")
		mainlog.With("envvar", "API_CONSUMER_TOKEN").Fatal(err)
		panic(err)
	}
	auth := env.Get("API_AUTH_TOKEN")
	if auth == "" {
		err := errors.New("required envvar missing")
		mainlog.With("envvar", "API_AUTH_TOKEN").Fatal(err)
		panic(err)
	}
	client, err = packet.NewClient(l, consumer, auth, apiBaseURL)
	if err != nil {
		mainlog.Fatal(err)
	}
	job.SetClient(client)
	job.SetProvisionerEngineName(provisionerEngineName)

	go func() {
		mainlog.With("addr", cfg.syslogAddr).Info("serving syslog")
		err = retry.Do(
			func() error {
				_, err := syslog.StartReceiver(cfg.syslogAddr, 1)

				return err
			},
		)
		if err != nil {
			mainlog.Fatal(errors.Wrap(err, "retry syslog serve"))
		}
	}()

	var nextServer net.IP
	var httpServerFQDN string
	g, ctx := errgroup.WithContext(ctx)
	ipxe := &ipxedust.Server{
		Log:                  defaultLogger(cfg.logLevel),
		EnableTFTPSinglePort: true,
		TFTP:                 ipxedust.ServerSpec{Disabled: true},
		HTTP:                 ipxedust.ServerSpec{Disabled: true},
	}

	if cfg.remoteTFTPAddr == "" { // use local iPXE binary service for TFTP
		if !cfg.iTFTPDisabled {
			ip := cfg.ipxe.TFTPAddr
			if strings.Contains(ip, ":") {
				ip = strings.Split(ip, ":")[0]
				if port := strings.Split(cfg.ipxe.TFTPAddr, ":")[1]; port != "69" {
					mainlog.With("providedPort", port).Info("warning: only port 69 is supported for TFTP")
				}
			}
			ipportTFTP, err := netaddr.ParseIPPort(ip + ":69")
			if err != nil {
				mainlog.Fatal(err)
			}
			ipxe.TFTP = ipxedust.ServerSpec{
				Addr:    ipportTFTP,
				Timeout: cfg.ipxe.TFTPTimeout,
			}
		}
		nextServer = conf.PublicIPv4
	} else { // use remote iPXE binary service for TFTP
		ip := net.ParseIP(cfg.remoteTFTPAddr)
		if ip == nil {
			mainlog.Fatal(errors.New("invalid remote TFTP server IP: " + cfg.remoteTFTPAddr))
		}
		nextServer = ip
		mainlog.With("addr", nextServer.String()).Info("serving iPXE binaries from remote TFTP server")
	}

	var ipxeHandler func(http.ResponseWriter, *http.Request)
	var ipxePattern string
	if cfg.remoteiHTTPAddr == "" { // use local iPXE binary service for HTTP
		if !cfg.iHTTPDisabled {
			ipxeHandler = ihttp.Handler{Log: defaultLogger(cfg.logLevel)}.Handle
		}
		ipxePattern = "/ipxe/"
		httpServerFQDN = cfg.httpAddr + ipxePattern
	} else { // use remote iPXE binary service for HTTP
		httpServerFQDN = cfg.remoteiHTTPAddr
		mainlog.With("addr", httpServerFQDN).Info("serving iPXE binaries from remote HTTP server")
	}
	g.Go(func() error {
		return ipxe.ListenAndServe(ctx)
	})

	mainlog.With("addr", cfg.dhcpAddr).Info("serving dhcp")
	go ServeDHCP(cfg.dhcpAddr, nextServer, httpServerFQDN)
	mainlog.With("addr", cfg.httpAddr).Info("serving http")
	go ServeHTTP(registerInstallers(), cfg.httpAddr, ipxePattern, ipxeHandler)

	<-ctx.Done()
	mainlog.Info("boots shutting down")
	err = g.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		mainlog.Fatal(err)
	}
}

// defaultLogger is zap logr implementation.
func defaultLogger(level string) logr.Logger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	zapLogger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}

	return zapr.NewLogger(zapLogger)
}

// customUsageFunc is a custom UsageFunc used for all commands
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
	fs.StringVar(&cfg.ipxe.TFTPAddr, "tftp-addr", "0.0.0.0", "local IP to listen on for serving iPXE binaries via TFTP.")
	fs.DurationVar(&cfg.ipxe.TFTPTimeout, "tftp-timeout", time.Second*5, "local iPXE TFTP server requests timeout.")
	fs.BoolVar(&cfg.iTFTPDisabled, "tftp-disabled", false, "disable serving iPXE binaries via TFTP.")
	fs.BoolVar(&cfg.iHTTPDisabled, "ihttp-disabled", false, "disable serving iPXE binaries via HTTP.")
	fs.StringVar(&cfg.remoteTFTPAddr, "remote-tftp-addr", "", "remote IP where iPXE binaries are served via TFTP. Overrides -tftp-addr.")
	fs.StringVar(&cfg.remoteiHTTPAddr, "remote-ihttp-addr", "", "remote IP and port where iPXE binaries are served via HTTP. Overrides -http-addr for iPXE binaries only.")
	fs.StringVar(&cfg.httpAddr, "http-addr", conf.HTTPBind, "local IP and port to listen on for the serving iPXE binaries and files via HTTP.")
	fs.StringVar(&cfg.logLevel, "log-level", "info", "log level.")
	fs.StringVar(&cfg.dhcpAddr, "dhcp-addr", conf.BOOTPBind, "IP and port to listen on for DHCP.")
	fs.StringVar(&cfg.syslogAddr, "syslog-addr", conf.SyslogBind, "IP and port to listen on for syslog messages.")

	return &ffcli.Command{
		Name:       name,
		ShortUsage: "Run Boots server for provisioning",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		UsageFunc: func(c *ffcli.Command) string {
			return customUsageFunc(c)
		},
	}
}

func registerInstallers() job.Installers {
	// register installers
	i := job.NewInstallers()
	// register coreos/flatcar
	c := coreos.Installer{}
	i.RegisterDistro("coreos", c.BootScript())
	i.RegisterDistro("flatcar", c.BootScript())
	// register custom ipxe
	ci := custom_ipxe.Installer{}
	i.RegisterDistro("custom_ipxe", ci.BootScript())
	i.RegisterInstaller("custom_ipxe", ci.BootScript())
	// register nixos
	n := nixos.Installer{Paths: nixos.BuildInitPaths()}
	i.RegisterDistro("nixos", n.BootScript())
	// register osie
	o := osie.Installer{}
	i.RegisterDistro("alpine", o.Rescue())
	i.RegisterDistro("discovery", o.Discover())
	// register osie as default
	d := osie.Installer{}
	i.RegisterDefaultInstaller(d.Install())
	// register rancher
	r := rancher.Installer{}
	i.RegisterDistro("rancher", r.BootScript())
	// register vmware
	v := vmware.Installer{}
	i.RegisterSlug("vmware_esxi_5_5", v.BootScriptVmwareEsxi55())
	i.RegisterSlug("vmware_esxi_6_0", v.BootScriptVmwareEsxi60())
	i.RegisterSlug("vmware_esxi_6_5", v.BootScriptVmwareEsxi65())
	i.RegisterSlug("vmware_esxi_6_7", v.BootScriptVmwareEsxi67())
	i.RegisterSlug("vmware_esxi_7_0", v.BootScriptVmwareEsxi70())
	i.RegisterSlug("vmware_esxi_7_0U2a", v.BootScriptVmwareEsxi70U2a())
	i.RegisterSlug("vmware_esxi_6_5_vcf", v.BootScriptVmwareEsxi65())
	i.RegisterSlug("vmware_esxi_6_7_vcf", v.BootScriptVmwareEsxi67())
	i.RegisterSlug("vmware_esxi_7_0_vcf", v.BootScriptVmwareEsxi70())
	i.RegisterDistro("vmware", v.BootScriptDefault())

	return i
}
