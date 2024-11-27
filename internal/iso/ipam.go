package iso

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/tinkerbell/smee/internal/dhcp/data"
)

func parseIPAM(d *data.DHCP) string {
	if d == nil {
		return ""
	}
	// return format is ipam=<mac-address>:<vlan-id>:<ip-address>:<netmask>:<gateway>:<hostname>:<dns>:<search-domains>:<ntp>
	ipam := make([]string, 9)
	ipam[0] = func() string {
		m := d.MACAddress.String()

		return strings.ReplaceAll(m, ":", "-")
	}()
	ipam[1] = func() string {
		if d.VLANID != "" {
			return d.VLANID
		}
		return ""
	}()
	ipam[2] = func() string {
		if d.IPAddress.Compare(netip.Addr{}) != 0 {
			return d.IPAddress.String()
		}
		return ""
	}()
	ipam[3] = func() string {
		if d.SubnetMask != nil {
			return net.IP(d.SubnetMask).String()
		}
		return ""
	}()
	ipam[4] = func() string {
		if d.DefaultGateway.Compare(netip.Addr{}) != 0 {
			return d.DefaultGateway.String()
		}
		return ""
	}()
	ipam[5] = d.Hostname
	ipam[6] = func() string {
		var nameservers []string
		for _, e := range d.NameServers {
			nameservers = append(nameservers, e.String())
		}
		if len(nameservers) > 0 {
			return strings.Join(nameservers, ",")
		}

		return ""
	}()
	ipam[7] = func() string {
		if len(d.DomainSearch) > 0 {
			return strings.Join(d.DomainSearch, ",")
		}

		return ""
	}()
	ipam[8] = func() string {
		var ntp []string
		for _, e := range d.NTPServers {
			ntp = append(ntp, e.String())
		}
		if len(ntp) > 0 {
			return strings.Join(ntp, ",")
		}

		return ""
	}()

	return fmt.Sprintf("ipam=%s", strings.Join(ipam, ":"))
}
