package main

import (
	"context"
	"errors"
	"flag"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/boots/http"
	"github.com/tinkerbell/dhcp/handler"
)

var startTime = time.Now()

type httpConfig struct {
	// addr is the address to listen on for the http server.
	addr string
	// extraKernelArgs are key=value pairs to be added as kernel commandline to the kernel in iPXE for OSIE.
	extraKernelArgs string
	// osieURL is the URL at which OSIE/Hook images live.
	osieURL string
	// tinkServerTLS is whether the tink server is using TLS.
	tinkServerTLS bool
	// tinkServerGRPCAddr is the address of the tink server.
	tinkServerGRPCAddr string
	// trustedProxies is a list of trusted proxies.
	trustedProxies []string
	// publicSyslogIP is the public IP address of the syslog server.
	publicSyslogIP string

	// ipxeVars are additional variable definitions to include in all iPXE installer
	// scripts. See https://ipxe.org/cfg. Separate multiple var definitions with spaces,
	// e.g. 'var1=val1 var2=val2'. Note that settings which require spaces (e.g, scriptlets)
	// are not yet supported.
	ipxeVars string
}

func (h *httpConfig) addFlags(fs *flag.FlagSet) {
	fs.StringVar(&h.ipxeVars, "ipxe-vars", "", "[http] additional variable definitions to include in all iPXE installer scripts. Separate multiple var definitions with spaces, e.g. 'var1=val1 var2=val2'.")
	fs.StringVar(&h.addr, "http-addr", "0.0.0.0:80", "[http] local IP and port to listen on for the serving iPXE binaries and files via HTTP.")
	fs.StringVar(&h.extraKernelArgs, "extra-kernel-args", "", "[http] Extra set of kernel args (k=v k=v) that are appended to the kernel cmdline when booting via iPXE.")
	fs.StringVar(&h.osieURL, "osie-url", "", "[http] URL where OSIE/Hook images are located.")
	fs.BoolVar(&h.tinkServerTLS, "tink-server-tls", false, "[http] Whether the tink server is using TLS.")
	fs.StringVar(&h.tinkServerGRPCAddr, "tink-server-grpc-addr", "", "[http] Address of the tink server.")
	fs.StringVar(&h.publicSyslogIP, "public-syslog-ip", "", "[http] Public IP address of the syslog server.")
	fs.Func("trusted-proxies", "[http] Comma-separated list of trusted proxies.", func(s string) error {
		var result []string
		for _, cidr := range strings.Split(s, ",") {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			_, _, err := net.ParseCIDR(cidr)
			if err != nil {
				// Its not a cidr, but maybe its an IP
				if ip := net.ParseIP(cidr); ip != nil {
					if ip.To4() != nil {
						cidr += "/32"
					} else {
						cidr += "/128"
					}
				} else {
					// not an IP
					return errors.New("invalid ip cidr in TRUSTED_PROXIES cidr=" + cidr)
				}
			}
			result = append(result, cidr)
		}
		h.trustedProxies = result

		return nil
	})
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
			OsieURL:            h.osieURL,
			ExtraKernelParams:  strings.Split(h.extraKernelArgs, " "),
			SyslogFQDN:         h.publicSyslogIP,
			TinkServerTLS:      h.tinkServerTLS,
			TinkServerGRPCAddr: h.tinkServerGRPCAddr,
		},
	}

	err := httpServer.ServeHTTP(ctx, h.addr, ipxeBinaryHandler)
	log.Info("shutting down http server")
	return err
}
