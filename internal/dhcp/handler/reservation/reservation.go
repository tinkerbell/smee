// Package reservation is the handler for responding to DHCPv4 messages with only host reservations.
package reservation

import (
	"net/netip"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/mvellasco/smee/internal/dhcp"
	"github.com/mvellasco/smee/internal/dhcp/handler"
)

// Handler holds the configuration details for the running the DHCP server.
type Handler struct {
	// Backend is the backend to use for getting DHCP data.
	Backend handler.BackendReader

	// IPAddr is the IP address to use in DHCP responses.
	// Option 54 and the sname DHCP header.
	// This could be a load balancer IP address or an ingress IP address or a local IP address.
	IPAddr netip.Addr

	// Log is used to log messages.
	// `logr.Discard()` can be used if no logging is desired.
	Log logr.Logger

	// Netboot configuration
	Netboot Netboot

	// OTELEnabled is used to determine if netboot options include otel naming.
	// When true, the netboot filename will be appended with otel information.
	// For example, the filename will be "snp.efi-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01".
	// <original filename>-00-<trace id>-<span id>-<trace flags>
	OTELEnabled bool

	// SyslogAddr is the address to send syslog messages to. DHCP Option 7.
	SyslogAddr netip.Addr
}

// Netboot holds the netboot configuration details used in running a DHCP server.
type Netboot struct {
	// iPXE binary server IP:Port serving via TFTP.
	IPXEBinServerTFTP netip.AddrPort

	// IPXEBinServerHTTP is the URL to the IPXE binary server serving via HTTP(s).
	IPXEBinServerHTTP *url.URL

	// IPXEScriptURL is the URL to the IPXE script to use.
	IPXEScriptURL func(*dhcpv4.DHCPv4) *url.URL

	// Enabled is whether to enable sending netboot DHCP options.
	Enabled bool

	// UserClass (for network booting) allows a custom DHCP option 77 to be used to break out of an iPXE loop.
	UserClass dhcp.UserClass
}
