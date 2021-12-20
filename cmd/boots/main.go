package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avast/retry-go"
	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/packethost/pkg/env"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	ipxe "github.com/tinkerbell/boots-ipxe"
	icmd "github.com/tinkerbell/boots-ipxe/cmd/ipxe"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/packet"
	"github.com/tinkerbell/boots/syslog"
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

func main() {
	httpAddr := flag.String("http-addr", conf.HTTPBind, "IP and port to listen on for HTTP.")
	c := icmd.Command{}
	flag.StringVar(&c.TFTPAddr, "tftp-addr", "0.0.0.0:69", "IP and port to listen on for serving iPXE binaries via TFTP.")
	rTFTP := flag.String("remote-tftp-addr", "", "IP where iPXE binaries are served via TFTP.")
	flag.DurationVar(&c.TFTPTimeout, "tftp-timeout", time.Second*5, "iPXE TFTP server timeout")
	flag.StringVar(&c.HTTPAddr, "ihttp-addr", "0.0.0.0:8080", "IP and port to listen on for serveing iPXE binaries via HTTP.")
	rHTTP := flag.String("remote-ihttp-addr", "", "IP and port where iPXE binaries are served via HTTP.")
	flag.DurationVar(&c.HTTPTimeout, "ihttp-timeout", time.Second*5, "iPXE HTTP server timeout")
	flag.Parse()

	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "boots")
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
		mainlog.Info("serving syslog")
		err = retry.Do(
			func() error {
				_, err := syslog.StartReceiver(1)

				return err
			},
		)
		if err != nil {
			mainlog.Fatal(errors.Wrap(err, "retry syslog serve"))
		}
	}()
	g, ctx := errgroup.WithContext(ctx)
	ipportHTTP, err := netaddr.ParseIPPort(c.HTTPAddr)
	if err != nil {
		mainlog.Fatal(err)
	}
	var nextServer net.IP
	var httpServerFQDN string
	if *rTFTP == "" && *rHTTP == "" {
		mainlog.Info("serving ipxe binaries via tftp and http")
		g.Go(func() error {
			ipportTFTP, err := netaddr.ParseIPPort(c.TFTPAddr)
			if err != nil {
				return err
			}
			s := ipxe.Server{
				TFTP: ipxe.ServerSpec{
					Addr:    ipportTFTP,
					Timeout: c.TFTPTimeout,
				},
				HTTP: ipxe.ServerSpec{
					Addr:    ipportHTTP,
					Timeout: c.HTTPTimeout,
				},
				Log: defaultLogger(flag.Lookup("log-level").Value.String()),
			}

			return s.ListenAndServe(ctx)
		})
		nextServer = conf.PublicIPv4
		httpServerFQDN = fmt.Sprintf("%v:%d", conf.PublicIPv4, ipportHTTP.Port())
	} else {
		nextServer = net.ParseIP(*rTFTP)
		httpServerFQDN = *rHTTP
	}
	mainlog.Info("serving dhcp")
	// ServeDHCP takes the next server address (nextServer), for serving the iPXE binaries via TFTP
	// and IP:Port (httpServerFQDN) for serving the iPXE binaries via HTTP.
	go ServeDHCP(nextServer, httpServerFQDN)
	mainlog.Info("serving http")
	go ServeHTTP(registerInstallers(), *httpAddr)

	<-ctx.Done()
	if *rTFTP == "" && *rHTTP == "" {
		err = g.Wait()
	}
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
