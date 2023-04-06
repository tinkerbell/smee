package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/dhcp"
	"github.com/tinkerbell/dhcp/handler/reservation"
)

type dhcpConfig struct {
	// listener is the local address for the DHCP server to listen on.
	listener netip.AddrPort
	enabled  bool
	handler  reservation.Handler
}

func (d *dhcpConfig) serveDHCP(ctx context.Context, log logr.Logger) error {
	listener := &dhcp.Listener{Addr: d.listener}
	d.handler.Log = log
	d.handler.Netboot.Enabled = true

	err := listener.ListenAndServe(ctx, &d.handler)
	log.Info("shutting down dhcp server")
	return err
}

func (d *dhcpConfig) String() string {
	if d.handler.Netboot.IPXEScriptURL != nil {
		return d.handler.Netboot.IPXEScriptURL.String()
	}
	return ""
}

func (d *dhcpConfig) Set(s string) error {
	if s == "" {
		return errors.New("ipxe-script-url cannot be empty")
	}
	if u, err := url.Parse(s); err != nil {
		return err
	} else {
		*d.handler.Netboot.IPXEScriptURL = *u
	}
	return nil
}

func (d *dhcpConfig) addFlags(fs *flag.FlagSet) {
	fs.BoolVar(&d.enabled, "dhcp-enabled", true, "[dhcp] enable DHCP service")
	fs.TextVar(&d.listener, "dhcp-bind-addr", netip.MustParseAddrPort("0.0.0.0:67"), "[dhcp] IP and port to bind and listen on for DHCP.")
	fs.TextVar(&d.handler.IPAddr, "dhcp-public-ip", autoDetectPublicIP(), "[dhcp] IP address from where clients can interact with DHCP (DHCP option 54).")
	fs.TextVar(&d.handler.Netboot.IPXEBinServerTFTP, "ipxe-tftp-addr", netip.AddrPort{}, "[dhcp][required] IP:Port where tftp clients can fetch iPXE binaries.")
	// fs.Var(d, "ipxe-script-url", "[dhcp] URL where clients can fetch the iPXE script.")
	fs.Func("ipxe-http-url", "[dhcp][required] URL where HTTP clients can fetch iPXE binaries.", func(s string) error {
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

	fs.Func("ipxe-script-url", "[dhcp] URL where clients can fetch the iPXE script.", func(s string) error {
		if s == "" {
			return fmt.Errorf("ipxe-script-url is required")
		}
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		d.handler.Netboot.IPXEScriptURL = u

		return nil
	})

}

func autoDetectPublicIP() netip.Addr {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return netip.Addr{}
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

		p, ok := netip.AddrFromSlice(v4.To4())
		if !ok {
			continue
		}

		return p
	}

	return netip.Addr{}
}

// required fields.
// d.Handler.IPAddr is required for DHCP option 54.
func (d *dhcpConfig) validate() error {
	if !d.enabled {
		return nil
	}
	if !d.listener.IsValid() {
		return errors.New("dhcp listener address is required")
	}
	if !d.handler.IPAddr.IsValid() {
		return errors.New("dhcp public IP address is required")
	}
	if d.handler.Netboot.IPXEScriptURL == nil {
		return errors.New("ipxe-script-url is required")
	}
	// if d.handler.Net

	return nil
}
