package standalone

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/backend"
)

// DiscoveryStandalone implements the Discovery interface for standalone operation.
type DiscoverStandalone struct {
	HardwareStandalone
}

func (ds *DiscoverStandalone) Instance() *backend.Instance {
	return ds.HardwareStandalone.Metadata.Instance
}

func (ds *DiscoverStandalone) MAC() net.HardwareAddr {
	m, err := net.ParseMAC(ds.getPrimaryInterface().DHCP.MAC)
	if err != nil {
		return nil
	}

	return m
}

// TODO: figure out where this gets used and how to return a useful value.
func (ds *DiscoverStandalone) Mode() string {
	return "hardware"
}

func (ds *DiscoverStandalone) GetIP(net.HardwareAddr) backend.IP {
	return ds.getPrimaryInterface().DHCP.IP
}

func (ds *DiscoverStandalone) GetMAC(ip net.IP) net.HardwareAddr {
	for _, iface := range ds.Network.Interfaces {
		if iface.DHCP.IP.Address.Equal(ip) {
			m, err := net.ParseMAC(iface.DHCP.MAC)
			if err != nil {
				return nil
			}

			return m
		}
	}

	// no way to return error so return an empty interface
	return nil
}

func (ds *DiscoverStandalone) DNSServers(net.HardwareAddr) []net.IP {
	iface := ds.getPrimaryInterface()
	out := make([]net.IP, len(iface.DHCP.NameServers))
	for i, v := range iface.DHCP.NameServers {
		out[i] = net.ParseIP(v)
	}

	return out
}

func (ds *DiscoverStandalone) LeaseTime(net.HardwareAddr) time.Duration {
	// TODO(@tobert) guessed that it's seconds, could be worng
	return time.Duration(ds.getPrimaryInterface().DHCP.LeaseTime) * time.Second
}

func (ds *DiscoverStandalone) Hostname() (string, error) {
	return ds.getPrimaryInterface().DHCP.Hostname, nil
}

func (ds *DiscoverStandalone) Hardware() backend.Hardware {
	return &ds.HardwareStandalone
}

func (ds *DiscoverStandalone) SetMAC(net.HardwareAddr) {
	// TODO: set the MAC, not sure this is useful?
}

// GetVLANID gets the VLAN ID for the given MAC address.
func (ds *DiscoverStandalone) GetVLANID(net.HardwareAddr) string {
	return ""
}
