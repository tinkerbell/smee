package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/avast/retry-go"
	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	dhcp4 "github.com/packethost/dhcp4-go"
	plog "github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/kubernetes"
	"github.com/tinkerbell/boots/client/standalone"
	"github.com/tinkerbell/boots/dhcp/server"
	"github.com/tinkerbell/boots/http"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/syslog"
	"github.com/tinkerbell/ipxedust"
	"github.com/tinkerbell/ipxedust/ihttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

var (
	// GitRev is the git revision of the build. It is set by the Makefile.
	GitRev = "unknown (use make)"

	startTime        = time.Now()
	publicIPv4       = mustPublicIPv4()
	publicFQDN       = getStringEnv("PUBLIC_FQDN", publicIPv4.String())
	publicSyslogIPv4 = mustPublicSyslogIPv4()
	publicSyslogFQDN = getStringEnv("PUBLIC_SYSLOG_FQDN", publicSyslogIPv4.String())
	syslogBind       = getStringEnv("SYSLOG_BIND", publicIPv4.String()+":514")
	httpBind         = getStringEnv("HTTP_BIND", publicIPv4.String()+":80")
	bootpBind        = getStringEnv("BOOTP_BIND", publicIPv4.String()+":67")
)

const name = "boots"

type config struct {
	// ipxe holds the config for serving ipxe binaries.
	ipxe ipxedust.Command
	// ipxeTFTPEnabled determines if local iPXE binaries served via TFTP are enabled.
	ipxeTFTPEnabled bool
	// ipxeHTTPEnabled determines if local iPXE binaries served via HTTP are enabled.
	ipxeHTTPEnabled bool
	// ipxeRemoteTFTPAddr is the address of the remote TFTP server serving iPXE binaries.
	ipxeRemoteTFTPAddr string
	// ipxeRemoteHTTPAddr is the address and port of the remote HTTP server serving iPXE binaries.
	ipxeRemoteHTTPAddr string
	// ipxeVars are additional variable definitions to include in all iPXE installer
	// scripts. See https://ipxe.org/cfg. Separate multiple var definitions with spaces,
	// e.g. 'var1=val1 var2=val2'. Note that settings which require spaces (e.g, scriptlets)
	// are not yet supported.
	ipxeVars string
	// httpAddr is the address of the HTTP server serving the iPXE script and other installer assets.
	httpAddr string
	// dhcpAddr is the local address for the DHCP server.
	dhcpAddr string
	// syslogAddr is the local address for the syslog server.
	syslogAddr string
	// loglevel is the log level for boots.
	logLevel string
	// extraKernelArgs are key=value pairs to be added as kernel commandline to the kernel in iPXE for OSIE.
	extraKernelArgs string
	// kubeConfig is the path to a kubernetes config file.
	kubeconfig string
	// kubeAPI is the Kubernetes API URL.
	kubeAPI string
	// kubeNamespace is an override for the namespace the kubernetes client will watch.
	kubeNamespace string
	// osieURL is the URL at which OSIE/Hook images live.
	osieURL string
	// iPXE script fragment to patch into binaries served over TFTP and HTTP.
	ipxeScriptPatch string
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

	go func() {
		log.Info("serving syslog", "addr", cfg.syslogAddr)
		err := retry.Do(
			func() error {
				_, err := syslog.StartReceiver(log, cfg.syslogAddr, 1)

				return err
			},
		)
		if err != nil {
			log.Error(err, "retry syslog serve")
			panic(errors.Wrap(err, "retry syslog serve"))
		}
	}()

	g, ctx := errgroup.WithContext(ctx)
	lg := log.WithValues("service", "github.com/tinkerbell/boots").WithName("github.com/tinkerbell/ipxedust")
	ipxe := &ipxedust.Server{
		Log:                  lg,
		EnableTFTPSinglePort: true,
		TFTP:                 ipxedust.ServerSpec{Disabled: true},
		HTTP:                 ipxedust.ServerSpec{Disabled: true},
	}
	var nextServer net.IP
	if cfg.ipxeRemoteTFTPAddr == "" { // use local iPXE binary service for TFTP
		if cfg.ipxeTFTPEnabled {
			ipportTFTP, err := netip.ParseAddrPort(cfg.ipxe.TFTPAddr)
			if err != nil {
				log.Error(err, "tftp addr must be an ip:port")
				panic(fmt.Errorf("%w: tftp addr must be an ip:port", err))
			}
			if ipportTFTP.Port() != 69 {
				err := fmt.Errorf("port for tftp addr must be 69, provided port: %d", ipportTFTP.Port())
				log.Error(err, "invalid port for tftp addr")
				panic(err)
			}
			ipxe.TFTP = ipxedust.ServerSpec{
				Addr:    ipportTFTP,
				Timeout: cfg.ipxe.TFTPTimeout,
				Patch:   []byte(cfg.ipxeScriptPatch),
			}
		}
		nextServer = publicIPv4
	} else { // use remote iPXE binary service for TFTP
		ip := net.ParseIP(cfg.ipxeRemoteTFTPAddr)
		if ip == nil {
			err := fmt.Errorf("invalid IP for remote TFTP server: %v", cfg.ipxeRemoteTFTPAddr)
			log.Error(err, "invalid IP for remote TFTP server")
			panic(err)
		}
		nextServer = ip
		log.Info("serving iPXE binaries from remote TFTP server", "addr", nextServer.String())
	}

	var ipxeHandler stdhttp.HandlerFunc
	var ipxePattern string
	var ipxeBaseURL string
	if cfg.ipxeRemoteHTTPAddr == "" { // use local iPXE binary service for HTTP
		if cfg.ipxeHTTPEnabled {
			ipxeHandler = ihttp.Handler{Log: lg, Patch: []byte(cfg.ipxeScriptPatch)}.Handle
		}
		ipxePattern = "/ipxe/"
		ipxeBaseURL = publicFQDN + ipxePattern
		log.Info("serving iPXE binaries from local HTTP server", "addr", ipxeBaseURL)
	} else { // use remote iPXE binary service for HTTP
		ipxeBaseURL = cfg.ipxeRemoteHTTPAddr
		log.Info("serving iPXE binaries from remote HTTP server", "addr", ipxeBaseURL)
	}
	g.Go(func() error {
		return ipxe.ListenAndServe(ctx)
	})

	finder, err := getHardwareFinder(log, cfg)
	if err != nil {
		log.Error(err, "get hardware finder")
		panic(err)
	}
	jobManager := job.NewCreator(log, finder)
	jobManager.DHCPServerIP = publicIPv4
	jobManager.PublicSyslogIPv4 = publicSyslogIPv4
	jobManager.Registry = getStringEnv("DOCKER_REGISTRY")
	jobManager.RegistryUsername = getStringEnv("REGISTRY_USERNAME")
	jobManager.RegistryPassword = getStringEnv("REGISTRY_PASSWORD")
	authority := getStringEnv("TINKERBELL_GRPC_AUTHORITY")
	if getStringEnv("DATA_MODEL_VERSION") == "1" && authority == "" {
		err := errors.New("TINKERBELL_GRPC_AUTHORITY env var is required when in tinkerbell mode (1)")
		log.Error(err, "TINKERBELL_GRPC_AUTHORITY env var is required when in tinkerbell mode (1)")
		panic(err)
	}

	osieURL, err := url.Parse(cfg.osieURL)
	if err != nil {
		log.Error(err, "osie url")
		panic(err)
	}
	httpServer := &http.Config{
		GitRev:             GitRev,
		StartTime:          startTime,
		Finder:             finder,
		Logger:             log,
		OSIEURL:            osieURL,
		ExtraKernelParams:  strings.Split(cfg.extraKernelArgs, " "),
		PublicSyslogFQDN:   publicSyslogFQDN,
		TinkServerTLS:      getBoolEnv("TINKERBELL_TLS", false),
		TinkServerGRPCAddr: authority,
		TrustedProxies:     parseTrustedProxies(),
	}

	dhcpServer := &server.Handler{
		JobManager: jobManager,
		Logger:     log,
		PoolSize:   getIntEnv("BOOTS_DHCP_WORKERS", runtime.GOMAXPROCS(0)/2),
	}

	log.Info("serving dhcp", "addr", cfg.dhcpAddr)
	// this flag.Set is needed to support how the log level is set in github.com/packethost/pkg/log
	_ = flag.Set("log-level", cfg.logLevel)

	// this is still need so that github.com/packethost/dhcp4-go doesn't panic
	l, err := plog.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	dhcp4.Init(l.Package("dhcp"))
	// bootsBaseURL is the hostname/ip + uri path to the http service serving iPXE binaries,
	// this is used when a netboot client identifies itself as HTTPClient.
	bootsBaseURL := publicFQDN
	go dhcpServer.ServeDHCP(cfg.dhcpAddr, nextServer, ipxeBaseURL, bootsBaseURL)

	log.Info("serving http", "addr", cfg.httpAddr)
	go httpServer.ServeHTTP(cfg.httpAddr, ipxePattern, ipxeHandler)

	<-ctx.Done()
	log.Info("boots shutting down")
	err = g.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "boots shutdown")
		panic(err)
	}
}

func getHardwareFinder(l logr.Logger, c *config) (client.HardwareFinder, error) {
	var hf client.HardwareFinder
	var err error

	switch os.Getenv("DATA_MODEL_VERSION") {
	case "standalone":
		saFile := os.Getenv("BOOTS_STANDALONE_JSON")
		if saFile == "" {
			return nil, errors.New("BOOTS_STANDALONE_JSON env must be set")
		}
		hf, err = standalone.NewHardwareFinder(saFile)
		if err != nil {
			return nil, err
		}
	case "kubernetes":
		kf, err := kubernetes.NewFinder(l, c.kubeAPI, c.kubeconfig, c.kubeNamespace)
		if err != nil {
			return nil, err
		}
		hf = kf
		// Start the client-side cache
		go func() {
			_ = kf.Start(context.Background())
		}()

	default:
		return nil, fmt.Errorf("must specify DATA_MODEL_VERSION")
	}

	return hf, nil
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
