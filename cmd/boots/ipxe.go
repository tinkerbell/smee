package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/ipxedust"
	"github.com/tinkerbell/ipxedust/ihttp"
)

type ipxeConfig struct {
	disableHTTP bool
	ipxedust.Server
}

func (i *ipxeConfig) addFlags(fs *flag.FlagSet) {
	i.addTFTPFlags(fs)
	i.addHTTPFlags(fs)
}

func (i *ipxeConfig) addTFTPFlags(fs *flag.FlagSet) {
	// tftp listener addr
	fs.Func("ipxe-tftp-addr", "[ipxe] local IP and port to listen on for serving iPXE binaries via TFTP (port must be 69).", func(s string) error {
		if s == "" {
			i.TFTP.Addr = netip.MustParseAddrPort("0.0.0.0:69")
			return nil
		}
		ipport, err := netip.ParseAddrPort(s)
		if err != nil {
			return err
		}
		i.TFTP.Addr = ipport

		return nil
	})
	// This sets the default value for the flag when coupled with fs.Func.
	_ = fs.Set("ipxe-tftp-addr", "0.0.0.0:69")

	// tftp timeout
	fs.DurationVar(&i.TFTP.Timeout, "ipxe-tftp-timeout", time.Second*5, "[ipxe] local iPXE TFTP server requests timeout.")

	// tftp disabled
	fs.BoolVar(&i.TFTP.Disabled, "ipxe-disable-tftp", false, "[ipxe] disable serving iPXE binaries via TFTP.")

	// tftp single port
	fs.BoolVar(&i.EnableTFTPSinglePort, "ipxe-enable-tftp-single-port", true, "[ipxe] enable serving iPXE binaries via TFTP on a single port instead of a random one.")

	// tftp patch
	fs.Func("ipxe-script-patch", "[ipxe] iPXE script fragment to patch into served iPXE binaries served via TFTP and HTTP.", func(s string) error {
		if s == "" {
			return nil
		}
		i.TFTP.Patch = []byte(s)
		return nil
	})

	// http is always disabled in this struct because we use the http.HandlerFunc from the github.com/tinkerbell/ipxedust library instead.
	i.HTTP.Disabled = true
}

func (i *ipxeConfig) addHTTPFlags(fs *flag.FlagSet) {
	fs.BoolVar(&i.disableHTTP, "ipxe-disable-http", false, "[ipxe] disable serving iPXE binaries via HTTP.")
}

func (i *ipxeConfig) binaryHandlerFunc(log logr.Logger) http.HandlerFunc {
	if i.disableHTTP {
		return nil
	}

	return ihttp.Handler{Log: log, Patch: i.TFTP.Patch}.Handle
}

func (i *ipxeConfig) tftpServer(log logr.Logger) (func(ctx context.Context) error, error) {
	if i.TFTP.Disabled {
		return nil, errors.New("iPXE tftp disabled")
	}
	i.Log = log
	return i.ListenAndServe, nil
}
