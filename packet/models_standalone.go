package packet

import (
	"net"
	"time"
)

// models_standalone.go contains a standalone backend for boots so it can run without
// a packet or tinkerbell backend. Instead of using a scalable backend, hardware data
// is loaded from a json file and stored in a list in memory.
//
// TODO:
//    * only supports one interface right now, multiple can be defined but might act weird
//    * methods only return the first interface's information and ignore everything else
//    * methods don't have godoc yet and since this is for test maybe never will?

// DiscoveryStandalone implements the Discovery interface for standalone operation
type DiscoverStandalone struct {
	*HardwareStandalone
}

// HardwareStandalone implements the Hardware interface for standalone operation
type HardwareStandalone struct {
	ID       string   `json:"id"`
	Network  Network  `json:"network"`
	Metadata Metadata `json:"metadata"`
}

// StandaloneClient is a placeholder for accessing data in []DiscoveryStandalone
// TODO: this probably doesn't need to be public?
// TODO: add the lookup methods for this (is there an interface somewhere?)
type StandaloneClient struct {
	filename string
	db       []DiscoverStandalone
}

func (ds DiscoverStandalone) Instance() *Instance {
	return ds.HardwareStandalone.Metadata.Instance
}

func (ds DiscoverStandalone) MAC() net.HardwareAddr {
	return ds.getPrimaryInterface().DHCP.MAC.HardwareAddr()
}

// TODO: figure out where this gets used and how to return a useful value
func (ds DiscoverStandalone) Mode() string {
	return "hardware"
}

func (ds DiscoverStandalone) GetIP(addr net.HardwareAddr) IP {
	return ds.getPrimaryInterface().DHCP.IP
}

func (ds DiscoverStandalone) GetMAC(ip net.IP) net.HardwareAddr {
	for _, iface := range ds.Network.Interfaces {
		if iface.DHCP.IP.Address.Equal(ip) {
			return iface.DHCP.MAC.HardwareAddr()
		}
	}

	// no way to return error so return an empty interface
	return ds.emptyInterface().DHCP.MAC.HardwareAddr()
}

func (ds DiscoverStandalone) DnsServers(mac net.HardwareAddr) []net.IP {
	iface := ds.getPrimaryInterface()
	out := make([]net.IP, len(iface.DHCP.NameServers))
	for i, v := range iface.DHCP.NameServers {
		out[i] = net.ParseIP(v)
	}

	return out
}

func (ds DiscoverStandalone) LeaseTime(mac net.HardwareAddr) time.Duration {
	// TODO(@tobert) guessed that it's seconds, could be worng
	return time.Duration(ds.getPrimaryInterface().DHCP.LeaseTime) * time.Second
}

func (ds DiscoverStandalone) Hostname() (string, error) {
	return ds.getPrimaryInterface().DHCP.Hostname, nil
}

func (ds DiscoverStandalone) Hardware() Hardware {
	var h Hardware = ds.HardwareStandalone

	return h
}

func (ds DiscoverStandalone) SetMAC(mac net.HardwareAddr) {
	// TODO: set the MAC, not sure this is useful?
}

func (hs HardwareStandalone) HardwareAllowPXE(mac net.HardwareAddr) bool {
	return hs.getPrimaryInterface().Netboot.AllowPXE
}

func (hs HardwareStandalone) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return hs.getPrimaryInterface().Netboot.AllowWorkflow
}

func (hs HardwareStandalone) HardwareArch(mac net.HardwareAddr) string {
	return hs.getPrimaryInterface().DHCP.Arch
}

func (hs HardwareStandalone) HardwareBondingMode() BondingMode {
	return hs.Metadata.BondingMode
}

func (hs HardwareStandalone) HardwareFacilityCode() string {
	return hs.Metadata.Facility.FacilityCode
}

func (hs HardwareStandalone) HardwareID() HardwareID {
	return HardwareID(hs.ID)
}

func (hs HardwareStandalone) HardwareIPs() []IP {
	out := make([]IP, len(hs.Network.Interfaces))
	for i, v := range hs.Network.Interfaces {
		out[i] = v.DHCP.IP
	}

	return out
}

func (hs HardwareStandalone) Interfaces() []Port {
	return []Port{} // stubbed out in tink too
}

func (hs HardwareStandalone) HardwareManufacturer() string {
	return hs.Metadata.Manufacturer.Slug
}

func (hs HardwareStandalone) HardwareProvisioner() string {
	return hs.Metadata.ProvisionerEngine
}

func (hs HardwareStandalone) HardwarePlanSlug() string {
	return hs.Metadata.Facility.PlanSlug
}

func (hs HardwareStandalone) HardwarePlanVersionSlug() string {
	return hs.Metadata.Facility.PlanVersionSlug
}

func (hs HardwareStandalone) HardwareState() HardwareState {
	return hs.Metadata.State
}

func (hs HardwareStandalone) HardwareOSIEVersion() string {
	return "" // stubbed out in tink too
}

func (hs HardwareStandalone) HardwareUEFI(mac net.HardwareAddr) bool {
	return hs.getPrimaryInterface().DHCP.UEFI
}

func (hs HardwareStandalone) OSIEBaseURL(mac net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.BaseURL
}

func (hs HardwareStandalone) KernelPath(mac net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.Kernel
}

func (hs HardwareStandalone) InitrdPath(mac net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.Initrd
}

func (hs HardwareStandalone) OperatingSystem() *OperatingSystem {
	return hs.Metadata.Instance.OS
}

// getPrimaryInterface returns the first interface in the list of interfaces
// and returns an empty interface with zeroed MAC & empty IP if that list is
// empty. Other models have more sophisticated interface selection but this
// model is mostly intended for test so this might behave in unexpected ways
// with more complex configurations and need more logic added.
func (hs HardwareStandalone) getPrimaryInterface() NetworkInterface {
	if len(hs.Network.Interfaces) >= 1 {
		return hs.Network.Interfaces[0]
	} else {
		return hs.emptyInterface()
	}
}

func (hs HardwareStandalone) emptyInterface() NetworkInterface {
	return NetworkInterface{
		DHCP: DHCP{
			MAC:         &MACAddr{},
			IP:          IP{},
			NameServers: []string{},
			TimeServers: []string{},
		},
		Netboot: Netboot{},
	}
}
