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
	"github.com/insomniacslk/dhcp/iana"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"github.com/tinkerbell/smee/internal/dhcp/otel"
)

// UserClass is DHCP option 77 (https://www.rfc-editor.org/rfc/rfc3004.html).
type UserClass string

// clientType is from DHCP option 60. Normally only PXEClient or HTTPClient.
type clientType string

const (
	pxeClient  clientType = "PXEClient"
	httpClient clientType = "HTTPClient"
)

// known user-class types. must correspond to DHCP option 77 - User-Class
// https://www.rfc-editor.org/rfc/rfc3004.html
const (
	// If the client has had iPXE burned into its ROM (or is a VM
	// that uses iPXE as the PXE "ROM"), special handling is
	// needed because in this mode the client is using iPXE native
	// drivers and chainloading to a UNDI stack won't work.
	IPXE UserClass = "iPXE"
	// If the client identifies as "Tinkerbell", we've already
	// chainloaded this client to the full-featured copy of iPXE
	// we supply. We have to distinguish this case so we don't
	// loop on the chainload step.
	Tinkerbell UserClass = "Tinkerbell"
)

// ArchToBootFile maps supported hardware PXE architectures types to iPXE binary files.
var ArchToBootFile = map[iana.Arch]string{
	iana.INTEL_X86PC:       "undionly.kpxe",
	iana.NEC_PC98:          "undionly.kpxe",
	iana.EFI_ITANIUM:       "undionly.kpxe",
	iana.DEC_ALPHA:         "undionly.kpxe",
	iana.ARC_X86:           "undionly.kpxe",
	iana.INTEL_LEAN_CLIENT: "undionly.kpxe",
	iana.EFI_IA32:          "ipxe.efi",
	iana.EFI_X86_64:        "ipxe.efi",
	iana.EFI_XSCALE:        "ipxe.efi",
	iana.EFI_BC:            "ipxe.efi",
	iana.EFI_ARM32:         "snp.efi",
	iana.EFI_ARM64:         "snp.efi",
	iana.EFI_X86_HTTP:      "ipxe.efi",
	iana.EFI_X86_64_HTTP:   "ipxe.efi",
	iana.EFI_ARM32_HTTP:    "snp.efi",
	iana.EFI_ARM64_HTTP:    "snp.efi",
	iana.Arch(41):          "snp.efi", // arm rpiboot: https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml#processor-architecture
}

// String function for clientType.
func (c clientType) String() string {
	return string(c)
}

// String function for UserClass.
func (u UserClass) String() string {
	return string(u)
}

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
		var opt60 string
		// if the client sends opt 60 with HTTPClient then we need to respond with opt 60
		if val := m.Options.Get(dhcpv4.OptionClassIdentifier); val != nil {
			if strings.HasPrefix(string(val), httpClient.String()) {
				d.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionClassIdentifier, []byte(httpClient)))
				opt60 = httpClient.String()
			}
		}
		d.BootFileName = "/netboot-not-allowed"
		d.ServerIPAddr = net.IPv4(0, 0, 0, 0)
		if n.AllowNetboot {
			a := arch(m)
			bin, found := ArchToBootFile[a]
			if !found {
				h.Log.Error(fmt.Errorf("unable to find bootfile for arch"), "network boot not allowed", "arch", a, "archInt", int(a), "mac", m.ClientHWAddr)
				return
			}
			uClass := UserClass(string(m.GetOneOption(dhcpv4.OptionUserClassInformation)))
			var ipxeScript *url.URL
			if h.Netboot.IPXEScriptURL != nil {
				ipxeScript = h.Netboot.IPXEScriptURL(m)
			}
			if n.IPXEScriptURL != nil {
				ipxeScript = n.IPXEScriptURL
			}
			d.BootFileName, d.ServerIPAddr = h.bootfileAndNextServer(ctx, uClass, opt60, bin, h.Netboot.IPXEBinServerTFTP, h.Netboot.IPXEBinServerHTTP, ipxeScript)
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
func (h *Handler) bootfileAndNextServer(ctx context.Context, uClass UserClass, opt60, bin string, tftp netip.AddrPort, ipxe, iscript *url.URL) (string, net.IP) {
	var nextServer net.IP
	var bootfile string
	if tp := otelhelpers.TraceparentStringFromContext(ctx); h.OTELEnabled && tp != "" {
		bin = fmt.Sprintf("%s-%v", bin, tp)
	}
	// If a machine is in an ipxe boot loop, it is likely to be that we aren't matching on IPXE or Tinkerbell userclass (option 77).
	switch { // order matters here.
	case uClass == Tinkerbell, (h.Netboot.UserClass != "" && uClass == h.Netboot.UserClass): // this case gets us out of an ipxe boot loop.
		bootfile = "/no-ipxe-script-defined"
		if iscript != nil {
			bootfile = iscript.String()
		}
	case clientType(opt60) == httpClient: // Check the client type from option 60.
		bootfile = ipxe.JoinPath(bin).String()
		nextServer = net.ParseIP("0.0.0.0")
		if n, err := netip.ParseAddrPort(ipxe.Host); err == nil {
			nextServer = n.Addr().AsSlice()
		} else if n2 := net.ParseIP(ipxe.Host); n2 != nil {
			nextServer = net.ParseIP(ipxe.Host)
		}
	case uClass == IPXE: // if the "iPXE" user class is found it means we aren't in our custom version of ipxe, but because of the option 43 we're setting we need to give a full tftp url from which to boot.
		bootfile = fmt.Sprintf("tftp://%v/%v", tftp.String(), bin)
		nextServer = net.IP(tftp.Addr().AsSlice())
	default:
		bootfile = bin
		nextServer = net.IP(tftp.Addr().AsSlice())
	}

	return bootfile, nextServer
}

// arch returns the arch of the client pulled from DHCP option 93.
func arch(d *dhcpv4.DHCPv4) iana.Arch {
	// get option 93 ; arch
	fwt := d.ClientArch()
	if len(fwt) == 0 {
		return iana.Arch(255) // unknown arch
	}
	var archKnown bool
	var a iana.Arch
	for _, elem := range fwt {
		if !strings.Contains(elem.String(), "unknown") {
			archKnown = true
			// Basic architecture identification, based purely on
			// the PXE architecture option.
			// https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml#processor-architecture
			a = elem
			break
		}
	}
	if !archKnown {
		return iana.Arch(255) // unknown arch
	}

	return a
}
