package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/netip"
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
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/kubernetes"
	"github.com/tinkerbell/boots/client/standalone"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/installers/customipxe"
	"github.com/tinkerbell/boots/installers/osie"
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
	provisionerEngineName = env.Get("PROVISIONER_ENGINE_NAME", "packet")

	mainlog log.Logger

	GitRev    = "unknown (use make)"
	StartTime = time.Now()
)

const name = "boots"

type config struct {
	// ipxe holds the config for serving ipxe binaries
	ipxe ipxedust.Command
	// ipxeTFTPEnabled determines if local iPXE binaries served via TFTP are enabled
	ipxeTFTPEnabled bool
	// ipxeHTTPEnabled determines if local iPXE binaries served via TFTP are enabled
	ipxeHTTPEnabled bool
	// ipxeRemoteTFTPAddr is the address of the remote TFTP server serving iPXE binaries
	ipxeRemoteTFTPAddr string
	// ipxeRemoteHTTPAddr is the address and port of the remote HTTP server serving iPXE binaries
	ipxeRemoteHTTPAddr string
	// ipxeVars are additional variable definitions to include in all iPXE installer
	// scripts. See https://ipxe.org/cfg. Separate multiple var definitions with spaces,
	// e.g. 'var1=val1 var2=val2'. Note that settings which require spaces (e.g, scriptlets)
	// are not yet supported.
	ipxeVars string
	// httpAddr is the address of the HTTP server serving the iPXE script and other installer assets
	httpAddr string
	// dhcpAddr is the local address for the DHCP server
	dhcpAddr string
	// syslogAddr is the local address for the syslog server
	syslogAddr string
	// loglevel is the log level for boots
	logLevel string
	// extraKernelArgs are key=value pairs to be added as kernel commandline to the kernel in iPXE for OSIE
	extraKernelArgs string
	// kubeConfig is the path to a kubernetes config file
	kubeconfig string
	// kubeAPI is the Kubernetes API URL
	kubeAPI string
	// kubeNamespace is an override for the namespace the kubernetes client will watch.
	kubeNamespace string
	// osiePathOverride allows a completely custom path/URL to be specified for OSIE/Hook images
	// This will bypass the hardcoded path appending of 'misc/osie/current' to the path
	osiePathOverride string
}

func main() {
	cfg := &config{}
	cli := newCLI(cfg, flag.NewFlagSet(name, flag.ExitOnError))
	_ = cli.Parse(os.Args[1:])

	// this flag.Set is needed to support how the log level is set in github.com/packethost/pkg/log
	_ = flag.Set("log-level", cfg.logLevel)
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

	workflowFinder, finder, err := getFinders(l, cfg)
	if err != nil {
		mainlog.Fatal(err)
	}
	jobManager := job.NewCreator(l, provisionerEngineName, finder)

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

	g, ctx := errgroup.WithContext(ctx)
	lg := defaultLogger(cfg.logLevel)
	lg = lg.WithValues("service", "github.com/tinkerbell/boots")
	lg = lg.WithName("github.com/tinkerbell/ipxedust")
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
				mainlog.Fatal(fmt.Errorf("%w: tftp addr must be an ip:port", err))
			}
			if ipportTFTP.Port() != 69 {
				mainlog.With("providedPort", ipportTFTP.Port()).Fatal(fmt.Errorf("port for tftp addr must be 69"))
			}
			ipxe.TFTP = ipxedust.ServerSpec{
				Addr:    ipportTFTP,
				Timeout: cfg.ipxe.TFTPTimeout,
			}
		}
		nextServer = conf.PublicIPv4
	} else { // use remote iPXE binary service for TFTP
		ip := net.ParseIP(cfg.ipxeRemoteTFTPAddr)
		if ip == nil {
			mainlog.Fatal(fmt.Errorf("invalid IP for remote TFTP server: %v", cfg.ipxeRemoteTFTPAddr))
		}
		nextServer = ip
		mainlog.With("addr", nextServer.String()).Info("serving iPXE binaries from remote TFTP server")
	}

	var ipxeHandler func(http.ResponseWriter, *http.Request)
	var ipxePattern string
	var ipxeBaseURL string
	bootsBaseURL := conf.PublicFQDN
	if cfg.ipxeRemoteHTTPAddr == "" { // use local iPXE binary service for HTTP
		if cfg.ipxeHTTPEnabled {
			ipxeHandler = ihttp.Handler{Log: lg}.Handle
		}
		ipxePattern = "/ipxe/"
		ipxeBaseURL = conf.PublicFQDN + ipxePattern
		mainlog.With("addr", ipxeBaseURL).Info("serving iPXE binaries from local HTTP server")
	} else { // use remote iPXE binary service for HTTP
		ipxeBaseURL = cfg.ipxeRemoteHTTPAddr
		mainlog.With("addr", ipxeBaseURL).Info("serving iPXE binaries from remote HTTP server")
	}
	g.Go(func() error {
		return ipxe.ListenAndServe(ctx)
	})

	httpServer := &BootsHTTPServer{
		finder:         finder,
		jobManager:     jobManager,
		workflowFinder: workflowFinder,
	}

	dhcpServer := &BootsDHCPServer{
		jobmanager: jobManager,
	}

	mainlog.With("addr", cfg.dhcpAddr).Info("serving dhcp")
	go dhcpServer.ServeDHCP(cfg.dhcpAddr, nextServer, ipxeBaseURL, bootsBaseURL)

	i, err := cfg.registerInstallers()
	if err != nil {
		mainlog.Fatal(err)
	}
	mainlog.With("addr", cfg.httpAddr).Info("serving http")
	go httpServer.ServeHTTP(i, cfg.httpAddr, ipxePattern, ipxeHandler)

	<-ctx.Done()
	mainlog.Info("boots shutting down")
	err = g.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		mainlog.Fatal(err)
	}
}

func getFinders(l log.Logger, c *config) (client.WorkflowFinder, client.HardwareFinder, error) {
	var hf client.HardwareFinder
	var wf client.WorkflowFinder = &client.NoOpWorkflowFinder{}
	var err error

	switch os.Getenv("DATA_MODEL_VERSION") {
	case "standalone":
		saFile := os.Getenv("BOOTS_STANDALONE_JSON")
		if saFile == "" {
			return nil, nil, errors.New("BOOTS_STANDALONE_JSON env must be set")
		}
		hf, err = standalone.NewHardwareFinder(saFile)
		if err != nil {
			return nil, nil, err
		}
		// standalone uses Tinkerbell workflows
		wf, err = standalone.NewWorkflowFinder()
		if err != nil {
			return nil, nil, err
		}
	case "kubernetes":
		kf, err := kubernetes.NewFinder(l, c.kubeAPI, c.kubeconfig, c.kubeNamespace)
		if err != nil {
			return nil, nil, err
		}
		wf = kf
		hf = kf
		// Start the client-side cache
		go func() {
			_ = kf.Start(context.Background())
		}()
	}

	return wf, hf, nil
}

// parseDynamicIPXEVars will parse any number of variable definitions from a
// string, and return a an array of two-element arrays which are the key/value
// string pairs of the variable's name and value. These will later be injected
// as variable definitions in the iPXE script. Variable order is preserved.
func parseDynamicIPXEVars(v string) ([][]string, error) {
	if v == "" {
		return nil, nil
	}

	// extract multiple variable definitions by splitting first on spaces
	allDefs := strings.Fields(v)
	retVal := make([][]string, len(allDefs))

	// iterate over each variable definition and split on '='
	for i, el := range allDefs {
		varDef := strings.SplitN(el, "=", 2)
		if len(varDef) == 2 && varDef[0] != "" && varDef[1] != "" {
			retVal[i] = varDef
		} else {
			err := errors.Errorf("unable to parse iPXE dynamic variable definitions from string %v", v)

			return nil, err
		}
	}

	return retVal, nil
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
	fs.StringVar(&cfg.httpAddr, "http-addr", conf.HTTPBind, "local IP and port to listen on for the serving iPXE binaries and files via HTTP.")
	fs.StringVar(&cfg.logLevel, "log-level", "info", "log level.")
	fs.StringVar(&cfg.dhcpAddr, "dhcp-addr", conf.BOOTPBind, "IP and port to listen on for DHCP.")
	fs.StringVar(&cfg.syslogAddr, "syslog-addr", conf.SyslogBind, "IP and port to listen on for syslog messages.")
	fs.StringVar(&cfg.extraKernelArgs, "extra-kernel-args", "", "Extra set of kernel args (k=v k=v) that are appended to the kernel cmdline when booting via iPXE.")
	fs.StringVar(&cfg.kubeconfig, "kubeconfig", "", "The Kubernetes config file location. Only applies if DATA_MODEL_VERSION=kubernetes.")
	fs.StringVar(&cfg.kubeAPI, "kubernetes", "", "The Kubernetes API URL, used for in-cluster client construction. Only applies if DATA_MODEL_VERSION=kubernetes.")
	fs.StringVar(&cfg.kubeNamespace, "kube-namespace", "", "An optional Kubernetes namespace override to query hardware data from.")
	fs.StringVar(&cfg.osiePathOverride, "osie-path-override", "", "A custom URL for OSIE/Hook images.")

	return &ffcli.Command{
		Name:       name,
		ShortUsage: "Run Boots server for provisioning",
		FlagSet:    fs,
		Options:    []ff.Option{ff.WithEnvVarPrefix(name)},
		UsageFunc:  customUsageFunc,
	}
}

func (cf *config) registerInstallers() (job.Installers, error) {
	// register installers
	i := job.NewInstallers()

	extraIPXEVars, err := parseDynamicIPXEVars(cf.ipxeVars)
	if err != nil {
		return job.Installers{}, err
	}

	// register custom ipxe
	o := customipxe.Installer(extraIPXEVars)
	i.RegisterDistro("custom_ipxe", o.BootScript("custom_ipxe"))
	i.RegisterInstaller("custom_ipxe", o.BootScript("custom_ipxe"))

	dataModelVersion := env.Get("DATA_MODEL_VERSION")
	auth := env.Get("TINKERBELL_GRPC_AUTHORITY")
	if dataModelVersion == "1" && auth == "" {
		return job.Installers{}, errors.New("TINKERBELL_GRPC_AUTHORITY is required when in tinkerbell mode")
	}

	// register osie
	o = osie.Installer(
		dataModelVersion,
		auth,
		cf.extraKernelArgs,
		env.Get("DOCKER_REGISTRY"),
		env.Get("REGISTRY_USERNAME"),
		env.Get("REGISTRY_PASSWORD"),
		env.Bool("TINKERBELL_TLS", true),
		cf.osiePathOverride,
		extraIPXEVars,
	)
	i.RegisterDistro("discovery", o.BootScript("discover"))
	i.RegisterDefaultInstaller(o.BootScript("default"))

	return i, nil
}
