package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/boots/http"
	"github.com/tinkerbell/dhcp/handler"
)

var startTime = time.Now()

type httpConfig struct {
	// addr is the address to listen on for the http server.
	addr netip.AddrPort
	// extraKernelArgs are key=value pairs to be added as kernel commandline to the kernel in iPXE for OSIE.
	extraKernelArgs []string
	// osieURL is the URL at which OSIE/Hook images live.
	osieURL *url.URL
	// tinkServerTLS is whether the tink server is using TLS.
	tinkServerTLS bool
	// tinkServerGRPCAddr is the address of the tink server.
	tinkServerGRPCAddr netip.AddrPort
	// trustedProxies is a list of trusted proxies.
	trustedProxies []string
	// publicSyslogIP is the public IP address of the syslog server.
	publicSyslogIP netip.Addr

	// ipxeVars are additional variable definitions to include in all iPXE installer
	// scripts. See https://ipxe.org/cfg. Separate multiple var definitions with commas,
	// e.g. 'var1=val1,var2=val2'. Note that settings which require spaces (e.g, scriptlets)
	// are not yet supported.
	ipxeVars []string
}

func (h *httpConfig) serveHTTP(ctx context.Context, log logr.Logger, ipxeBinaryHandler stdhttp.HandlerFunc, finder handler.BackendReader) error {
	httpServer := &http.Config{
		GitRev:         GitRev,
		StartTime:      startTime,
		Logger:         log,
		TrustedProxies: h.trustedProxies,
		IPXEScript: &http.IPXEScript{
			Finder:             finder,
			Logger:             log,
			OsieURL:            h.osieURL.String(),
			ExtraKernelParams:  h.extraKernelArgs,
			SyslogFQDN:         h.publicSyslogIP.String(),
			TinkServerTLS:      h.tinkServerTLS,
			TinkServerGRPCAddr: h.tinkServerGRPCAddr.String(),
		},
	}

	fmt.Printf("httpServer: %+v\n", httpServer.IPXEScript)

	err := httpServer.ServeHTTP(ctx, h.addr.String(), ipxeBinaryHandler)
	log.Info("shutting down http server")
	return err
}

func (h *httpConfig) addFlags(fs *flag.FlagSet) {
	fs.TextVar(&h.addr, "http-addr", netip.MustParseAddrPort("0.0.0.0:80"), "[http] local IP and port to listen on for the serving iPXE binaries and files via HTTP.")
	fs.BoolVar(&h.tinkServerTLS, "tink-server-tls", false, "[http] Whether the tink server is using TLS.")
	fs.TextVar(&h.tinkServerGRPCAddr, "tink-server-grpc-addr", netip.AddrPort{}, "[http] Address of the tink server.")
	fs.TextVar(&h.publicSyslogIP, "public-syslog-ip", netip.Addr{}, "[http] Public IP address of the syslog server.")
	fs.Func("ipxe-vars", "[http] additional variable definitions to include in all iPXE installer scripts. Separate multiple var definitions with commas, e.g. 'var1=val1,var2=val2'.", func(s string) error {
		h.ipxeVars = strings.Split(s, ",")
		return nil
	})
	fs.Func("extra-kernel-args", "[http] Extra set of kernel args (k=v,k=v) that are appended to the kernel cmdline when booting via iPXE.", func(s string) error {
		h.extraKernelArgs = strings.Split(s, ",")
		return nil
	})
	fs.Func("osie-url", "[http] URL where OSIE/Hook images are located.", func(s string) error {
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		h.osieURL = u
		return nil
	})
	fs.Func("trusted-proxies", "[http] Comma-separated list of trusted proxy cidrs (e.g. '172.16.2.1/24,172.16.3.1/24').", func(s string) error {
		r, err := ipNetSliceConv(s)
		if err != nil {
			return err
		}
		for _, elem := range r {
			h.trustedProxies = append(h.trustedProxies, elem.String())
		}
		return nil
	})
}

func ipNetSliceConv(val string) ([]net.IPNet, error) {
	val = strings.Trim(val, "[]")
	// Emtpy string would cause a slice with one (empty) entry
	if len(val) == 0 {
		return []net.IPNet{}, nil
	}
	ss := strings.Split(val, ",")
	out := make([]net.IPNet, len(ss))
	for i, sval := range ss {
		_, n, err := net.ParseCIDR(strings.TrimSpace(sval))
		if err != nil {
			return nil, fmt.Errorf("invalid string being converted to CIDR: %s", sval)
		}
		out[i] = *n
	}
	return out, nil
}

func (h *httpConfig) validate() error {
	if !h.addr.IsValid() {
		return fmt.Errorf("http-addr must be set")
	}
	if h.osieURL == nil {
		return fmt.Errorf("osie-url must be set")
	}
	return nil
}
