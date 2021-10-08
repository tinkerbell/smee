package main

import (
	"context"
	"flag"
	"time"

	"github.com/equinix-labs/otel-init-go/otelinit"
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
	"github.com/tinkerbell/boots/tftp"

	"github.com/tinkerbell/boots/installers/coreos"
	"github.com/tinkerbell/boots/installers/custom_ipxe"
	"github.com/tinkerbell/boots/installers/nixos"
	"github.com/tinkerbell/boots/installers/osie"
	"github.com/tinkerbell/boots/installers/rancher"
	"github.com/tinkerbell/boots/installers/vmware"

	"github.com/avast/retry-go"
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
	flag.Parse()

	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")

	ctx := context.Background()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "boots")
	defer otelShutdown(ctx)

	metrics.Init(l)
	dhcp.Init(l)
	conf.Init(l)
	httplog.Init(l)
	installers.Init(l)
	job.Init(l)
	syslog.Init(l)
	tftp.Init(l)
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

	mainlog.Info("serving tftp")
	go ServeTFTP()
	mainlog.Info("serving dhcp")
	go ServeDHCP()

	mainlog.Info("serving http")
	ServeHTTP(registerInstallers())
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
