package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/boots/ipxe/bhttp"
	"github.com/tinkerbell/boots/ipxe/script"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/syslog"
	"github.com/tinkerbell/dhcp"
	"github.com/tinkerbell/dhcp/handler"
	"github.com/tinkerbell/dhcp/handler/reservation"
	"github.com/tinkerbell/ipxedust"
	"github.com/tinkerbell/ipxedust/ihttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

var (
	// GitRev is the git revision of the build. It is set by the Makefile.
	GitRev = "unknown (use make)"

	startTime = time.Now()
)

const name = "boots"

type config struct {
	syslog         syslogConfig
	tftp           tftp
	ipxeHTTPBinary ipxeHTTPBinary
	ipxeHTTPScript ipxeHTTPScript
	dhcp           dhcpConfig

	// loglevel is the log level for boots.
	logLevel string
	backends dhcpBackends
}

type syslogConfig struct {
	enabled  bool
	bindAddr string
}

type tftp struct {
	enabled         bool
	bindAddr        string
	timeout         time.Duration
	ipxeScriptPatch string
}

type ipxeHTTPBinary struct {
	enabled bool
}

type ipxeHTTPScript struct {
	enabled          bool
	bindAddr         string
	extraKernelArgs  string
	trustedProxies   string
	hookURL          string
	tinkServer       string
	tinkServerUseTLS bool
}

type dhcpConfig struct {
	enabled           bool
	bindAddr          string
	ipForPacket       string
	syslogIP          string
	tftpIP            string
	httpIpxeBinaryIP  string
	httpIpxeScriptURL string
}

type dhcpBackends struct {
	file       backend.File
	kubernetes backend.Kube
}

func main() {
	cfg := &config{}
	cli := newCLI(cfg, flag.NewFlagSet(name, flag.ExitOnError))
	_ = cli.Parse(os.Args[1:])

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, name)
	defer otelShutdown(ctx)
	metrics.Init()

	log := defaultLogger(cfg.logLevel)
	log.Info("starting", "version", GitRev)

	g, ctx := errgroup.WithContext(ctx)
	// syslog
	if cfg.syslog.enabled {
		log.Info("starting syslog server", "bind_addr", cfg.syslog.bindAddr)
		g.Go(func() error {
			if err := syslog.StartReceiver(ctx, log, cfg.syslog.bindAddr, 1); err != nil {
				log.Error(err, "syslog server failure")
				return err
			}
			<-ctx.Done()
			log.Info("syslog server stopped")
			return nil
		})
	}

	tftpServer := &ipxedust.Server{
		Log:                  log.WithValues("service", "github.com/tinkerbell/boots").WithName("github.com/tinkerbell/ipxedust"),
		HTTP:                 ipxedust.ServerSpec{Disabled: true}, // disabled because below we use the http handlerfunc instead.
		EnableTFTPSinglePort: true,
	}
	// tftp
	if cfg.tftp.enabled {
		tftpServer.EnableTFTPSinglePort = true
		if ip, err := netip.ParseAddrPort(cfg.tftp.bindAddr); err != nil {
			log.Error(err, "invalid bind address")
			panic(errors.Wrap(err, "invalid bind address"))
		} else {
			tftpServer.TFTP = ipxedust.ServerSpec{
				Disabled: false,
				Addr:     ip,
				Timeout:  cfg.tftp.timeout,
				Patch:    []byte(cfg.tftp.ipxeScriptPatch),
			}
			// start the ipxe binary tftp server
			log.Info("starting tftp server", "bind_addr", cfg.tftp.bindAddr)
			g.Go(func() error {
				return tftpServer.ListenAndServe(ctx)
			})
		}
	}

	handlers := map[string]http.HandlerFunc{}
	// http ipxe binaries
	if cfg.ipxeHTTPBinary.enabled {
		handlers["/ipxe/"] = ihttp.Handler{
			Log:   log.WithValues("service", "github.com/tinkerbell/boots").WithName("github.com/tinkerbell/ipxedust"),
			Patch: []byte(cfg.tftp.ipxeScriptPatch),
		}.Handle
	}

	// http ipxe script
	if cfg.ipxeHTTPScript.enabled {
		f := static{}
		switch {
		case cfg.backends.file.Enabled:
			b, err := cfg.backends.file.Backend(ctx, log)
			if err != nil {
				panic(fmt.Errorf("failed to run file backend: %w", err))
			}
			f.backend = b
		default: // default backend is kubernetes
			b, err := cfg.backends.kubernetes.Backend(ctx)
			if err != nil {
				panic(fmt.Errorf("failed to run kubernetes backend: %w", err))
			}
			f.backend = b
		}

		jh := script.Handler{
			Logger:             log,
			Finder:             f,
			OSIEURL:            cfg.ipxeHTTPScript.hookURL,
			ExtraKernelParams:  strings.Split(cfg.ipxeHTTPScript.extraKernelArgs, " "),
			PublicSyslogFQDN:   cfg.dhcp.syslogIP,
			TinkServerTLS:      cfg.ipxeHTTPScript.tinkServerUseTLS,
			TinkServerGRPCAddr: cfg.ipxeHTTPScript.tinkServer,
		}
		handlers["/"] = jh.HandlerFunc()
	}

	// start the http server for ipxe binaries and scripts
	httpServer := &bhttp.Config{
		GitRev:         GitRev,
		StartTime:      startTime,
		Logger:         log,
		TrustedProxies: parseTrustedProxies(cfg.ipxeHTTPScript.trustedProxies),
	}
	log.Info("serving http", "addr", cfg.ipxeHTTPScript.bindAddr)
	g.Go(func() error {
		return httpServer.ServeHTTP(ctx, cfg.ipxeHTTPScript.bindAddr, handlers)
	})

	// dhcp server
	if cfg.dhcp.enabled {
		listener, dh, err := cfg.dhcpListener(ctx, log)
		if err != nil {
			log.Error(err, "failed to create dhcp listener")
			panic(errors.Wrap(err, "failed to create dhcp listener"))
		}
		log.Info("starting dhcp server", "bind_addr", cfg.dhcp.bindAddr)
		g.Go(func() error {
			return listener.ListenAndServe(ctx, dh)
		})
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "failed running all Boots services")
		panic(err)
	}
	log.Info("boots is shutting down")
}

func (c *config) dhcpListener(ctx context.Context, log logr.Logger) (*dhcp.Listener, *reservation.Handler, error) {
	// 1. create the handler
	// 2. create the backend
	// 3. add the backend to the handler
	// 4. create the listener
	// 5. start the listener(handler)
	pktIP, err := netip.ParseAddr(c.dhcp.ipForPacket)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid bind address: %w", err)
	}
	tftpIP, err := netip.ParseAddrPort(c.dhcp.tftpIP)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid tftp address for DHCP server: %w", err)
	}
	httpBinaryURL, err := url.Parse(c.dhcp.httpIpxeBinaryIP)
	if err != nil || httpBinaryURL == nil {
		return nil, nil, fmt.Errorf("invalid http ipxe binary url: %w", err)
	}
	httpScriptURL, err := url.Parse(c.dhcp.httpIpxeScriptURL)
	if err != nil || httpScriptURL == nil {
		return nil, nil, fmt.Errorf("invalid http ipxe script url: %w", err)
	}
	syslogIP, err := netip.ParseAddr(c.dhcp.syslogIP)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid syslog address: %w", err)
	}
	dh := &reservation.Handler{
		Backend: nil,
		IPAddr:  pktIP,
		Log:     log,
		Netboot: reservation.Netboot{
			IPXEBinServerTFTP: tftpIP,
			IPXEBinServerHTTP: httpBinaryURL,
			IPXEScriptURL:     httpScriptURL,
			Enabled:           true,
		},
		OTELEnabled: true,
		SyslogAddr:  syslogIP,
	}
	switch {
	case c.backends.file.Enabled:
		b, err := c.backends.file.Backend(ctx, log)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create file backend: %w", err)
		}
		dh.Backend = b
	default: // default backend is kubernetes
		b, err := c.backends.kubernetes.Backend(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create kubernetes backend: %w", err)
		}
		dh.Backend = b
	}
	bindAddr, err := netip.ParseAddrPort(c.dhcp.bindAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid tftp address for DHCP server: %w", err)
	}

	return &dhcp.Listener{Addr: bindAddr}, dh, nil
}

type static struct {
	backend handler.BackendReader
}

func (s static) Find(ctx context.Context, ip net.IP) (script.Data, error) {
	d, n, err := s.backend.GetByIP(ctx, ip)
	if err != nil {
		return script.Data{}, err
	}

	return script.Data{
		AllowNetboot:  n.AllowNetboot,
		Console:       "",
		MACAddress:    d.MACAddress,
		Arch:          d.Arch,
		VLANID:        d.VLANID,
		WorkflowID:    d.MACAddress.String(),
		Facility:      n.Facility,
		IPXEScript:    n.IPXEScript,
		IPXEScriptURL: n.IPXEScriptURL,
	}, nil
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
