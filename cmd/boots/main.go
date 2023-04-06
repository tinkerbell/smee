package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/syslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

const name = "boots"

// GitRev is the git revision of the build. It is set by the Makefile.
var GitRev = "unknown (use make)"

type config struct {
	// loglevel is the log level for Boots.
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
		fmt.Printf("error parsing cli, %v\n", err)
		os.Exit(1)
	}
	if err := cfg.validateAll(fs); err != nil {
		fmt.Printf("error validating config: %v\n", err)
		os.Exit(2)
	}

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, name)
	defer otelShutdown(ctx)
	metrics.Init()

	log := defaultLogger(cfg.logLevel).WithName("github.com/tinkerbell/boots")
	log.Info("starting", "version", GitRev)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if cfg.syslogEnabled {
			log.Info("starting syslog server", "addr", cfg.syslogAddr)
			// TODO: validate the config
			_, err := syslog.StartReceiver(log, cfg.syslogAddr, 1)
			return err
		}

		return nil
	})

	g.Go(func() error {
		log.Info("starting tftp server", "addr", cfg.ipxe.TFTP.Addr)
		// TODO: validate the config
		fn, err := cfg.ipxe.tftpServer(log)
		if err != nil {
			return err
		}
		return fn(ctx)
	})

	g.Go(func() error {
		log.Info("starting http server", "addr", cfg.http.addr /*"ipxeURL", ipxeBaseURL*/)
		// TODO: validate the config
		finder, err := getBackend(ctx, log, cfg)
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
	log.Info("Boots shutting down")
	keysAndValues := []interface{}{}
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		keysAndValues = append(keysAndValues, "err", err)
	}
	log.Info("Boots shutdown", keysAndValues...)
}

func newCLI(cfg *config, fs *flag.FlagSet) *ffcli.Command {
	fs.StringVar(&cfg.logLevel, "log-level", "info", "log level.")
	fs.BoolVar(&cfg.syslogEnabled, "syslog-enabled", true, "[syslog] Enable syslog server.")
	fs.StringVar(&cfg.syslogAddr, "syslog-addr", "", "[syslog] IP and port to listen on for syslog messages.")
	// fs.StringVar(&cfg.backend, "backend", "kubernetes", "The backend to use for retrieving hardware data. Valid options are: standalone, kubernetes.")
	fs.Func("backend", "The backend to use for retrieving hardware data. Valid options are: standalone, kubernetes.", func(s string) error {
		switch strings.ToLower(s) {
		case "kubernetes":
			cfg.k8s.enabled = true
		case "standalone":
			cfg.standalone.enabled = true
		default:
			return errors.New("invalid backend")
		}

		return nil
	})
	fs.StringVar(&cfg.k8s.config, "kubeconfig", "", "[k8s] The Kubernetes config file location. Only applies if backend is kubernetes.")
	fs.StringVar(&cfg.k8s.api, "kubeapi", "", "[k8s] The Kubernetes API URL, used for in-cluster client construction. Only applies if backend is kubernetes.")
	fs.StringVar(&cfg.k8s.namespace, "kubenamespace", "", "[k8s] A Kubernetes namespace from which hardware data is queried.")
	fs.StringVar(&cfg.standalone.file, "standalone-file", "", "[standalone] The path to a JSON file containing hardware data. Only applies if backend is standalone.")
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

func formatErr(err, e error) error {
	if err != nil {
		err = fmt.Errorf("\n%w\n%w", e, err)
	} else {
		err = fmt.Errorf("\n%w", e)
	}

	return err
}

func (c *config) validateAll(fs *flag.FlagSet) error {
	if !c.k8s.enabled && !c.standalone.enabled {
		c.k8s.enabled = true
	}
	var err error
	if e := c.ipxe.validate(); e != nil {
		err = formatErr(err, e)
	}

	if e := c.dhcp.validate(); e != nil {
		err = formatErr(err, e)
	}

	if e := c.http.validate(); e != nil {
		err = formatErr(err, e)
	}

	if e := c.k8s.validate(fs); e != nil {
		err = formatErr(err, e)
	}

	if e := c.standalone.validate(fs); e != nil {
		err = formatErr(err, e)
	}

	if err != nil {
		fs.Usage()
	}

	return err
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
