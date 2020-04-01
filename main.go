package main

import (
	"flag"
	"net"
	"time"

	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/env"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/packet"
	"github.com/tinkerbell/boots/syslog"
	"github.com/tinkerbell/boots/tftp"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"

	_ "github.com/tinkerbell/boots/installers/coreos"
	_ "github.com/tinkerbell/boots/installers/custom_ipxe"
	_ "github.com/tinkerbell/boots/installers/nixos"
	_ "github.com/tinkerbell/boots/installers/osie"
	_ "github.com/tinkerbell/boots/installers/rancher"
	_ "github.com/tinkerbell/boots/installers/vmware"

	"github.com/avast/retry-go"
)

var (
	client     *packet.Client
	apiBaseURL = env.DefaultURL("API_BASE_URL", "https://api.packet.net")

	mainlog log.Logger

	GitRev    = "unknown (use make)"
	StartTime = time.Now()
)

func parseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, len(cidrs))
	for i := range cidrs {
		_, net, err := net.ParseCIDR(cidrs[i])
		if err != nil {
			return nil, errors.Wrap(err, "parsing CIDR string")
		}
		nets[i] = net
	}
	return nets, nil
}

func main() {
	flag.Parse()

	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")
	dhcp.Init(l)
	env.Init(l)
	httplog.Init(l)
	installers.Init(l)
	job.Init(l)
	syslog.Init(l)
	tftp.Init(l)
	mainlog.With("version", GitRev).Info("starting")

	client, err = packet.NewClient(env.Require("API_CONSUMER_TOKEN"), env.Require("API_AUTH_TOKEN"), apiBaseURL)
	if err != nil {
		mainlog.Fatal(err)
	}
	job.SetClient(client)

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
	ServeHTTP()
}
