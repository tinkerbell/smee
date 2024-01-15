// package main is an example of how to use the dhcp package with the kube backend.
package main

import (
	"context"
	"log"
	"net"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/equinix-labs/otel-init-go/otelinit"
	"github.com/go-logr/stdr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/tinkerbell/smee/backend/kube"
	"github.com/tinkerbell/smee/dhcp"
	"github.com/tinkerbell/smee/dhcp/handler"
	"github.com/tinkerbell/smee/dhcp/handler/reservation"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	ctx, otelShutdown := otelinit.InitOpenTelemetry(ctx, "github.com/tinkerbell/smee/dhcp")
	defer otelShutdown(ctx)

	l := stdr.New(log.New(os.Stdout, "", log.Lshortfile))
	l = l.WithName("github.com/tinkerbell/smee/dhcp")
	// 1. create the backend
	// 2. create the handler(backend)
	// 3. create the listener(handler)
	backend, err := kubeBackend(ctx)
	if err != nil {
		panic(err)
	}

	h := &reservation.Handler{
		Log:    l,
		IPAddr: netip.MustParseAddr("192.168.2.50"),
		Netboot: reservation.Netboot{
			IPXEBinServerTFTP: netip.MustParseAddrPort("192.168.2.50:69"),
			IPXEBinServerHTTP: &url.URL{Scheme: "http", Host: "192.168.2.50:8080"},
			IPXEScriptURL: func(*dhcpv4.DHCPv4) *url.URL {
				return &url.URL{Scheme: "http", Host: "192.168.2.50", Path: "auto.ipxe"}
			},
			Enabled: true,
		},
		OTELEnabled: true,
		Backend:     backend,
	}
	conn, err := server4.NewIPv4UDPConn("", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("0.0.0.0:67")))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = conn.Close()
	}()
	server := &dhcp.Server{Logger: l, Conn: conn, Handlers: []dhcp.Handler{h}}
	l.Info("starting server", "addr", h.IPAddr)
	l.Error(server.Serve(ctx), "done")
	l.Info("done")
}

func kubeBackend(ctx context.Context) (handler.BackendReader, error) {
	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: "/home/tink/.kube/config",
		},
		&clientcmd.ConfigOverrides{
			Context: api.Context{
				Namespace: "tink-system",
			},
		},
	)

	config, err := ccfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	k, err := kube.NewBackend(config)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = k.Start(ctx)
	}()

	return k, nil
}
