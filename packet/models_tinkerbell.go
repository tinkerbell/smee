package packet

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/conf"
)

func (i InterfaceTinkerbell) Name() string {
	return i.DHCP.IfaceName
}

func (d DiscoveryTinkerbell) LeaseTime(mac net.HardwareAddr) time.Duration {
	return d.Network.InterfaceByMac(mac).DHCP.LeaseTime
}

func (d DiscoveryTinkerbell) Hardware() *Hardware {
	var h Hardware = d.HardwareTinkerbell
	return &h
}

func (d DiscoveryTinkerbell) DnsServers() []net.IP {
	var servers = conf.DNSServers
	// change to new way

	return servers
}

func (d DiscoveryTinkerbell) Instance() *Instance {
	return d.Metadata.Instance
}

func (d DiscoveryTinkerbell) Mac() net.HardwareAddr {
	return d.mac
}

func (d DiscoveryTinkerbell) Mode() string {
	return "hardware"
}

func (d DiscoveryTinkerbell) GetIp(mac net.HardwareAddr) IP {
	//if i := d.Network.Interface(mac); i.DHCP.IP.Address != nil {
	//	return i.DHCP.IP
	//}
	return d.Network.InterfaceByMac(mac).DHCP.IP
}

func (d DiscoveryTinkerbell) GetMac(ip net.IP) net.HardwareAddr {
	return d.Network.InterfaceByIp(ip).DHCP.MAC.HardwareAddr()
}

func (n Network) InterfaceByMac(mac net.HardwareAddr) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.MAC.String() == mac.String() {
			return i
		}
	}
	return n.Default // if there's no default then it'd be empty anyway?
}

func (n Network) InterfaceByIp(ip net.IP) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.IP.Address.String() == ip.String() {
			return i
		}
	}
	return n.Default // if there's no default then it'd be empty anyway?
}

func (d *DiscoveryTinkerbell) PrimaryDataMAC() MACAddr {
	mac := OnesMAC
	// TODO
	return mac
}

func (d *DiscoveryTinkerbell) Hostname() (string, error) {
	return d.Instance().Hostname, nil // temp
}

func (d *DiscoveryTinkerbell) SetMac(mac net.HardwareAddr) {
	d.mac = mac
}

func (h HardwareTinkerbell) HardwareAllowPXE(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).Netboot.AllowPXE
}

func (h HardwareTinkerbell) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).Netboot.AllowWorkflow
}

func (h HardwareTinkerbell) HardwareArch(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).DHCP.Arch
}

func (h HardwareTinkerbell) HardwareBondingMode() BondingMode {
	return h.Metadata.BondingMode
}

func (h HardwareTinkerbell) HardwareFacilityCode() string {
	return h.Metadata.Facility.FacilityCode
}

func (h HardwareTinkerbell) HardwareID() string {
	return h.ID
}

func (h HardwareTinkerbell) HardwareIPs() []IP {
	var hips []IP
	// TODO
	return hips
}

//func (h HardwareTinkerbell) HardwareIPMI() net.IP {
//	return h.DHCP.IP // is this correct?
//}

func (h HardwareTinkerbell) HardwareManufacturer() string {
	return h.Metadata.Manufacturer.Slug
}

func (h HardwareTinkerbell) HardwarePlanSlug() string {
	return h.Metadata.Facility.PlanSlug
}

func (h HardwareTinkerbell) HardwarePlanVersionSlug() string {
	return h.Metadata.Facility.PlanVersionSlug
}

func (h HardwareTinkerbell) HardwareState() HardwareState {
	return h.Metadata.State
}

// dummy method for backward compatibility
func (h HardwareTinkerbell) HardwareServicesVersion() string {
	return ""
}

func (h HardwareTinkerbell) HardwareUEFI(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).DHCP.UEFI
}

func (h HardwareTinkerbell) Interfaces() []Port {
	var ports []Port
	return ports
}

func (h HardwareTinkerbell) OsieBaseURL(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.BaseURL
}

func (h HardwareTinkerbell) KernelPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.Kernel
}

func (h HardwareTinkerbell) InitrdPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.Initrd
}
