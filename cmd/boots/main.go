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
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/packet"
	"github.com/tinkerbell/boots/syslog"
	"github.com/tinkerbell/ipxedust"
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
	c := ipxedust.Command{}
	flag.StringVar(&c.TFTPAddr, "tftp-addr", "0.0.0.0:69", "local IP and port to listen on for serving iPXE binaries via TFTP.")
	flag.StringVar(&c.HTTPAddr, "ihttp-addr", "0.0.0.0:8080", "local IP and port to listen on for serving iPXE binaries via HTTP.")
	flag.DurationVar(&c.TFTPTimeout, "tftp-timeout", time.Second*5, "local iPXE TFTP server requests timeout")
	flag.DurationVar(&c.HTTPTimeout, "ihttp-timeout", time.Second*5, "local iPXE HTTP server requests timeout")
	tftpDisabled := flag.Bool("tftp-disabled", false, "disable serving iPXE binaries via TFTP")
	ihttpDisabled := flag.Bool("ihttp-disabled", false, "disable serving iPXE binaries via HTTP")
	rTFTP := flag.String("remote-tftp-addr", "", "remote IP and port where iPXE binaries are served via TFTP.")
	rHTTP := flag.String("remote-ihttp-addr", "", "remote IP and port where iPXE binaries are served via HTTP.")
	httpAddr := flag.String("http-addr", conf.HTTPBind, "local IP and port to listen on for the serving iPXE files via HTTP.")

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

	var nextServer net.IP
	var httpServerFQDN string
	g, ctx := errgroup.WithContext(ctx)
	ipxe := &ipxedust.Server{
		Log:                  defaultLogger(flag.Lookup("log-level").Value.String()),
		EnableTFTPSinglePort: true,
		TFTP:                 ipxedust.ServerSpec{Disabled: true},
		HTTP:                 ipxedust.ServerSpec{Disabled: true},
	}

	if *rTFTP == "" { // use local iPXE binary service for TFTP
		if !*tftpDisabled {
			ipportTFTP, err := netaddr.ParseIPPort(c.TFTPAddr)
			if err != nil {
				mainlog.Fatal(err)
			}
			ipxe.TFTP = ipxedust.ServerSpec{
				Addr:    ipportTFTP,
				Timeout: c.TFTPTimeout,
			}
		}
		nextServer = conf.PublicIPv4
	} else { // use remote iPXE binary service for TFTP
		// TODO(jacobweinstock): validate input
		nextServer = net.ParseIP(*rTFTP)
	}

	if *rHTTP == "" { // use local iPXE binary service for HTTP
		if !*ihttpDisabled {
			ipportHTTP, err := netaddr.ParseIPPort(c.HTTPAddr)
			if err != nil {
				mainlog.Fatal(err)
			}
			ipxe.HTTP = ipxedust.ServerSpec{
				Addr:    ipportHTTP,
				Timeout: c.HTTPTimeout,
			}
			httpServerFQDN = fmt.Sprintf("%v:%d", conf.PublicIPv4, ipportHTTP.Port())
		} else {
			httpServerFQDN = conf.PublicIPv4.String()
		}
	} else { // use remote iPXE binary service for HTTP
		httpServerFQDN = *rHTTP
	}
	g.Go(func() error {
		return ipxe.ListenAndServe(ctx)
	})

	mainlog.Info("serving dhcp")
	// ServeDHCP takes the next server address (nextServer), for serving the iPXE binaries via TFTP
	// and IP:Port (httpServerFQDN) for serving the iPXE binaries via HTTP.
	go ServeDHCP(nextServer, httpServerFQDN)
	mainlog.Info("serving http")
	go ServeHTTP(registerInstallers(), *httpAddr)

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
