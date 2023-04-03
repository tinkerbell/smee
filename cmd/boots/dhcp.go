package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/tinkerbell/dhcp"
	"github.com/tinkerbell/dhcp/backend/kube"
	"github.com/tinkerbell/dhcp/handler/reservation"
	"golang.org/x/sync/errgroup"
	"inet.af/netaddr"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type dhcpConfig struct {
	// listener is the local address for the DHCP server to listen on.
	listener netaddr.IPPort
	enabled  bool
	handler  reservation.Handler
}

func (d *dhcpConfig) serveDHCP(ctx context.Context, log logr.Logger) error {
	listener := &dhcp.Listener{Addr: d.listener}
	d.handler.Log = log
	d.handler.Netboot.Enabled = true

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		e := listener.ListenAndServe(&d.handler)
		return e
	})
	<-ctx.Done()
	listener.Shutdown()
	err := g.Wait()
	log.Info("shutting down dhcp server")
	return err
}

func (k *k8sConfig) kubeBackend(ctx context.Context) (reservation.BackendReader, error) {
	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: k.config,
		},
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: k.api,
			},
			Context: clientcmdapi.Context{
				Namespace: k.namespace,
			},
		},
	)

	config, err := ccfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	kb, err := kube.NewBackend(config)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = kb.Start(ctx)
	}()

	return kb, nil
}

func (d *dhcpConfig) addFlags(fs *flag.FlagSet) {
	fs.BoolVar(&d.enabled, "dhcp-enabled", true, "[dhcp] enable DHCP service")
	fs.Func("dhcp-addr", "[dhcp] IP and port to listen on for DHCP.", func(s string) error {
		if s == "" {
			d.listener = netaddr.MustParseIPPort("0.0.0.0:67")

			return nil
		}
		v, err := netaddr.ParseIPPort(s)
		if err != nil {
			return err
		}
		d.listener = v

		return nil
	})
	// This sets the default value for the flag when coupled with fs.Func.
	fs.Set("dhcp-addr", "0.0.0.0:67")

	fs.Func("dhcp-public-ip", "[dhcp] public IP address where Boots will be available. Used for DHCP option 54", func(s string) error {
		var p netaddr.IP
		if s == "" || s == "0.0.0.0" {
			var err error
			p, err = autoDetectPublicIP()
			if err != nil {
				return fmt.Errorf("'-public-ip', unable to auto-detect: %v", err)
			}
		} else {
			var err error
			p, err = netaddr.ParseIP(s)
			if err != nil {
				return fmt.Errorf("'-public-ip', invalid address: %v", s)
			}
		}

		d.handler.IPAddr = p
		d.handler.Netboot.IPXEBinServerTFTP = netaddr.IPPortFrom(p, 69)
		d.handler.Netboot.IPXEBinServerHTTP = &url.URL{Scheme: "http", Host: p.String()}
		d.handler.Netboot.IPXEScriptURL = &url.URL{Scheme: "http", Host: p.String(), Path: "/auto.ipxe"}
		return nil
	})
	fs.Set("dhcp-public-ip", "0.0.0.0")
	fs.Func("ipxe-remote-tftp-addr", "[dhcp] remote IP:Port where iPXE binaries are served via TFTP.", func(s string) error {
		if s == "" {
			return nil
		}
		v, err := netaddr.ParseIPPort(s)
		if err != nil {
			return err
		}
		d.handler.Netboot.IPXEBinServerTFTP = v

		return nil
	})
	fs.Func("ipxe-remote-http-addr", "[dhcp] remote URL where iPXE binaries are served via HTTP.", func(s string) error {
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil {
			fmt.Println("error parsing url", s, err)
			return err
		}
		d.handler.Netboot.IPXEBinServerHTTP = u

		return nil
	})
	fs.Func("ipxe-script-url", "[dhcp] remote URL where iPXE script is served.", func(s string) error {
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		d.handler.Netboot.IPXEScriptURL = u

		return nil
	})
}

func autoDetectPublicIP() (netaddr.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		err = errors.Wrap(err, "unable to auto-detect public IPv4")
		return netaddr.IP{}, err
	}
	for _, addr := range addrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ip.IP.To4()
		if v4 == nil || !v4.IsGlobalUnicast() {
			continue
		}

		p, ok := netaddr.FromStdIP(v4)
		if !ok {
			continue
		}

		return p, nil
	}

	return netaddr.IP{}, errors.New("unable to auto-detect public IPv4")
}
