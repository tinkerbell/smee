package reservation

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/equinix-labs/otel-init-go/otelhelpers"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/tinkerbell/smee/internal/dhcp"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"github.com/tinkerbell/smee/internal/dhcp/otel"
)

// setDHCPOpts takes a client dhcp packet and data (typically from a backend) and creates a slice of DHCP packet modifiers.
// m is the DHCP request from a client. d is the data to use to create the DHCP packet modifiers.
// This is most likely the place where we would have any business logic for determining DHCP option setting.
func (h *Handler) setDHCPOpts(_ context.Context, _ *dhcpv4.DHCPv4, d *data.DHCP) []dhcpv4.Modifier {
	mods := []dhcpv4.Modifier{
		dhcpv4.WithLeaseTime(d.LeaseTime),
		dhcpv4.WithYourIP(d.IPAddress.AsSlice()),
	}
	if len(d.NameServers) > 0 {
		mods = append(mods, dhcpv4.WithDNS(d.NameServers...))
	}
	if len(d.DomainSearch) > 0 {
		mods = append(mods, dhcpv4.WithDomainSearchList(d.DomainSearch...))
	}
	if len(d.NTPServers) > 0 {
		mods = append(mods, dhcpv4.WithOption(dhcpv4.OptNTPServers(d.NTPServers...)))
	}
	if d.BroadcastAddress.Compare(netip.Addr{}) != 0 {
		mods = append(mods, dhcpv4.WithGeneric(dhcpv4.OptionBroadcastAddress, d.BroadcastAddress.AsSlice()))
	}
	if d.DomainName != "" {
		mods = append(mods, dhcpv4.WithGeneric(dhcpv4.OptionDomainName, []byte(d.DomainName)))
	}
	if d.Hostname != "" {
		mods = append(mods, dhcpv4.WithGeneric(dhcpv4.OptionHostName, []byte(d.Hostname)))
	}
	if len(d.SubnetMask) > 0 {
		mods = append(mods, dhcpv4.WithNetmask(d.SubnetMask))
	}
	if d.DefaultGateway.Compare(netip.Addr{}) != 0 {
		mods = append(mods, dhcpv4.WithRouter(d.DefaultGateway.AsSlice()))
	}
	if h.SyslogAddr.Compare(netip.Addr{}) != 0 {
		mods = append(mods, dhcpv4.WithOption(dhcpv4.OptGeneric(dhcpv4.OptionLogServer, h.SyslogAddr.AsSlice())))
	}

	return mods
}

// setNetworkBootOpts purpose is to sets 3 or 4 values. 2 DHCP headers, option 43 and optionally option (60).
// These headers and options are returned as a dhcvp4.Modifier that can be used to modify a dhcp response.
// github.com/insomniacslk/dhcp uses this method to simplify packet manipulation.
//
// DHCP Headers (https://datatracker.ietf.org/doc/html/rfc2131#section-2)
// 'siaddr': IP address of next bootstrap server. represented below as `.ServerIPAddr`.
// 'file': Client boot file name. represented below as `.BootFileName`.
//
// DHCP option
// option 60: Class Identifier. https://www.rfc-editor.org/rfc/rfc2132.html#section-9.13
// option 60 is set if the client's option 60 (Class Identifier) starts with HTTPClient.
func (h *Handler) setNetworkBootOpts(ctx context.Context, m *dhcpv4.DHCPv4, n *data.Netboot) dhcpv4.Modifier {
	// m is a received DHCPv4 packet.
	// d is the reply packet we are building.
	withNetboot := func(d *dhcpv4.DHCPv4) {
		// if the client sends opt 60 with HTTPClient then we need to respond with opt 60
		if val := m.Options.Get(dhcpv4.OptionClassIdentifier); val != nil {
			if strings.HasPrefix(string(val), dhcp.HTTPClient.String()) {
				d.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionClassIdentifier, []byte(dhcp.HTTPClient)))
			}
		}
		d.BootFileName = "/netboot-not-allowed"
		d.ServerIPAddr = net.IPv4(0, 0, 0, 0)
		if n.AllowNetboot {
			i := dhcp.NewInfo(m)
			if i.IPXEBinary == "" {
				return
			}
			uClass := dhcp.UserClass(string(m.GetOneOption(dhcpv4.OptionUserClassInformation)))
			var ipxeScript *url.URL
			if h.Netboot.IPXEScriptURL != nil {
				ipxeScript = h.Netboot.IPXEScriptURL(m)
			}
			if n.IPXEScriptURL != nil {
				ipxeScript = n.IPXEScriptURL
			}
			d.BootFileName, d.ServerIPAddr = h.bootfileAndNextServer(ctx, m, uClass, h.Netboot.IPXEBinServerTFTP, h.Netboot.IPXEBinServerHTTP, ipxeScript)
			pxe := dhcpv4.Options{ // FYI, these are suboptions of option43. ref: https://datatracker.ietf.org/doc/html/rfc2132#section-8.4
				// PXE Boot Server Discovery Control - bypass, just boot from filename.
				6:  []byte{8},
				69: otel.TraceparentFromContext(ctx),
			}
			d.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionVendorSpecificInformation, pxe.ToBytes()))
		}
	}

	return withNetboot
}

// bootfileAndNextServer returns the bootfile (string) and next server (net.IP).
// input arguments `tftp`, `ipxe` and `iscript` use non string types so as to attempt to be more clear about the expectation around what is wanted for these values.
// It also helps us avoid having to validate a string in multiple ways.
func (h *Handler) bootfileAndNextServer(ctx context.Context, pkt *dhcpv4.DHCPv4, uClass dhcp.UserClass, tftp netip.AddrPort, ipxe, iscript *url.URL) (string, net.IP) {
	var nextServer net.IP
	var bootfile string
	i := dhcp.NewInfo(pkt)
	if tp := otelhelpers.TraceparentStringFromContext(ctx); h.OTELEnabled && tp != "" {
		i.IPXEBinary = fmt.Sprintf("%s-%v", i.IPXEBinary, tp)
	}
	nextServer = i.NextServer(ipxe, tftp)
	bootfile = i.Bootfile(uClass, iscript, ipxe, tftp)

	return bootfile, nextServer
}
