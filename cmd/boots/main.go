package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/boots/backend/kubernetes"
	"github.com/tinkerbell/boots/backend/standalone"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

var (
	// GitRev is the git revision of the build. It is set by the Makefile.
	GitRev = "unknown (use make)"
)

const name = "boots"

type config struct {
	// loglevel is the log level for boots.
	logLevel      string
	syslogEnabled bool
	// syslogAddr is the local address to listen on for the syslog server.
	syslogAddr string
	// backend to use for retrieving hardware data.
	backend    string
	k8s        *k8sConfig
	standalone *standaloneConfig
	dhcp       *dhcpConfig
	ipxe       *ipxeConfig
	http       *httpConfig
}

type k8sConfig struct {
	// config is the path to a kubernetes config file.
	config string
	// api is the Kubernetes API URL.
	api string
	// namespace is an override for the namespace the kubernetes client will watch.
	namespace string
}

type standaloneConfig struct {
	// file is the path to a JSON file containing hardware data.
	file string
}

func main() {
	cfg := &config{
		k8s:        &k8sConfig{},
		standalone: &standaloneConfig{},
		dhcp:       &dhcpConfig{},
		ipxe:       &ipxeConfig{},
		http:       &httpConfig{},
	}
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cli := newCLI(cfg, fs)
	if err := cli.Parse(os.Args[1:]); err != nil {
		//fmt.Printf("type %T\n", err)
		//fmt.Printf("error parsing cli, %v\n", err)
		os.Exit(1)
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, name)
	defer otelShutdown(ctx)
	metrics.Init()

	log := defaultLogger(cfg.logLevel)
	log.Info("starting", "version", GitRev)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if cfg.syslogEnabled {
			log.Info("starting syslog server", "addr", cfg.syslogAddr)
			_, err := syslog.StartReceiver(log, cfg.syslogAddr, 1)
			return err
		}

		return nil
	})

	g.Go(func() error {
		log.Info("starting tftp server", "addr", cfg.ipxe.TFTP.Addr)
		lg := log.WithValues("service", "github.com/tinkerbell/boots").WithName("github.com/tinkerbell/ipxedust")
		fn, err := cfg.ipxe.tftpServer(lg)
		if err != nil {
			return nil
		}
		return fn(ctx)
	})

	g.Go(func() error {
		log.Info("starting http server", "addr", cfg.http.addr /*"ipxeURL", ipxeBaseURL*/)
		finder, err := getHardwareFinder(log, cfg)
		if err != nil {
			log.Info("error getting hardware finder", "err", err)

			return err
		}
		return cfg.http.serveHTTP(ctx, log, cfg.ipxe.binaryHandlerFunc(log), finder)
	})

	g.Go(func() error {
		if cfg.dhcp.enabled {
			backend, err := cfg.k8s.kubeBackend(ctx)
			if err != nil {
				return errors.New("failed to create kubernetes backend")
			}
			log.Info("starting dhcp server", "addr", cfg.dhcp.listener)
			cfg.dhcp.handler.Backend = backend
			// TODO: validate the config
			return cfg.dhcp.serveDHCP(ctx, log)
		}

		return nil
	})

	<-ctx.Done()
	log.Info("boots shutting down")
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Info("boots shutdown", "err", err)
	}
}

func newCLI(cfg *config, fs *flag.FlagSet) *ffcli.Command {
	fs.StringVar(&cfg.logLevel, "log-level", "info", "log level.")
	fs.BoolVar(&cfg.syslogEnabled, "syslog-enabled", true, "[syslog] Enable syslog server.")
	fs.StringVar(&cfg.syslogAddr, "syslog-addr", "", "[syslog] IP and port to listen on for syslog messages.")
	fs.StringVar(&cfg.backend, "backend", "kubernetes", "The backend to use for retrieving hardware data. Valid options are: standalone, kubernetes.")
	fs.StringVar(&cfg.k8s.config, "kubeconfig", "", "The Kubernetes config file location. Only applies if backend is kubernetes.")
	fs.StringVar(&cfg.k8s.api, "kubeapi", "", "The Kubernetes API URL, used for in-cluster client construction. Only applies if backend is kubernetes.")
	fs.StringVar(&cfg.k8s.namespace, "kubenamespace", "", "An optional Kubernetes namespace override to query hardware data from.")
	fs.StringVar(&cfg.standalone.file, "standalone-file", "", "The path to a JSON file containing hardware data. Only applies if backend is standalone.")
	cfg.ipxe.addFlags(fs)
	cfg.dhcp.addFlags(fs)
	cfg.http.addFlags(fs)

	return &ffcli.Command{
		Name:       name,
		ShortUsage: "Run Boots server for provisioning",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		Exec: func(context.Context, []string) error {
			return nil
		},
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

func getHardwareFinder(l logr.Logger, c *config) (backend.HardwareFinder, error) {
	var hf backend.HardwareFinder
	var err error

	switch c.backend {
	case "standalone":
		hf, err = standalone.NewHardwareFinder(c.standalone.file)
		if err != nil {
			return nil, err
		}
	case "kubernetes":
		kf, err := kubernetes.NewFinder(l, c.k8s.api, c.k8s.config, c.k8s.namespace)
		if err != nil {
			return nil, err
		}
		hf = kf
		// Start the client-side cache
		go func() {
			_ = kf.Start(context.Background())
		}()

	default:
		return nil, fmt.Errorf("must specify either standalone or kubernetes backend")
	}

	return hf, nil
}
