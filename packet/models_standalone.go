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
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return ds.Network.Interfaces[0].DHCP.MAC.HardwareAddr()
}

func (ds DiscoverStandalone) Mode() string {
	return "testing"
}

func (ds DiscoverStandalone) GetIP(addr net.HardwareAddr) IP {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return ds.Network.Interfaces[0].DHCP.IP
}

func (ds DiscoverStandalone) GetMAC(ip net.IP) net.HardwareAddr {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return ds.Network.Interfaces[0].DHCP.MAC.HardwareAddr()
}

func (ds DiscoverStandalone) DnsServers(mac net.HardwareAddr) []net.IP {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	out := make([]net.IP, len(ds.Network.Interfaces[0].DHCP.NameServers))
	for i, v := range ds.Network.Interfaces[0].DHCP.NameServers {
		out[i] = net.ParseIP(v)
	}
	return out
}

func (ds DiscoverStandalone) LeaseTime(mac net.HardwareAddr) time.Duration {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	// TODO(@tobert) guessed that it's seconds, could be worng
	return time.Duration(ds.Network.Interfaces[0].DHCP.LeaseTime) * time.Second
}

func (ds DiscoverStandalone) Hostname() (string, error) {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return ds.Network.Interfaces[0].DHCP.Hostname, nil
}

func (ds DiscoverStandalone) Hardware() Hardware {
	var h Hardware = ds.HardwareStandalone
	return h
}

func (ds DiscoverStandalone) SetMAC(mac net.HardwareAddr) {
	if len(ds.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	// TODO: set the MAC, not sure this is useful?
}

func (hs HardwareStandalone) HardwareAllowPXE(mac net.HardwareAddr) bool {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].Netboot.AllowPXE
}

func (hs HardwareStandalone) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].Netboot.AllowWorkflow
}

func (hs HardwareStandalone) HardwareArch(mac net.HardwareAddr) string {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].DHCP.Arch
}

func (hs HardwareStandalone) HardwareBondingMode() BondingMode {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Metadata.BondingMode
}

func (hs HardwareStandalone) HardwareFacilityCode() string {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Metadata.Facility.FacilityCode
}

func (hs HardwareStandalone) HardwareID() HardwareID {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
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
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].DHCP.UEFI
}

func (hs HardwareStandalone) OSIEBaseURL(mac net.HardwareAddr) string {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].Netboot.OSIE.BaseURL
}

func (hs HardwareStandalone) KernelPath(mac net.HardwareAddr) string {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].Netboot.OSIE.Kernel
}

func (hs HardwareStandalone) InitrdPath(mac net.HardwareAddr) string {
	if len(hs.Network.Interfaces) < 1 {
		panic("not enough interfaces in json-defined host")
	}
	return hs.Network.Interfaces[0].Netboot.OSIE.Initrd
}

func (hs HardwareStandalone) OperatingSystem() *OperatingSystem {
	return hs.Metadata.Instance.OS
}
