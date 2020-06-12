package packet

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/conf"
)

func (i InterfaceTinkerbell) Name() string {
	return i.DHCP.IfaceName
}

func (d DiscoveryTinkerbellV1) LeaseTime(mac net.HardwareAddr) time.Duration {
	return d.Network.InterfaceByMac(mac).DHCP.LeaseTime
}

func (d DiscoveryTinkerbellV1) Hardware() *Hardware {
	var h Hardware = d.HardwareTinkerbellV1
	return &h
}

func (d DiscoveryTinkerbellV1) DnsServers() []net.IP {
	var servers = conf.DNSServers
	// change to new way

	return servers
}

func (d DiscoveryTinkerbellV1) Instance() *Instance {
	return d.Metadata.Instance
}

func (d DiscoveryTinkerbellV1) Mac() net.HardwareAddr {
	return d.mac
}

func (d DiscoveryTinkerbellV1) Mode() string {
	return "hardware"
}

func (d DiscoveryTinkerbellV1) GetIp(mac net.HardwareAddr) IP {
	//if i := d.Network.Interface(mac); i.DHCP.IP.Address != nil {
	//	return i.DHCP.IP
	//}
	return d.Network.InterfaceByMac(mac).DHCP.IP
}

func (d DiscoveryTinkerbellV1) GetMac(ip net.IP) net.HardwareAddr {
	return d.Network.InterfaceByIp(ip).DHCP.MAC.HardwareAddr()
}

func (n Network) InterfaceByMac(mac net.HardwareAddr) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.MAC.String() == mac.String() {
			return i
		}
	}
	return NetworkInterface{}
}

func (n Network) InterfaceByIp(ip net.IP) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.IP.Address.String() == ip.String() {
			return i
		}
	}
	return NetworkInterface{}
}

func (d *DiscoveryTinkerbellV1) PrimaryDataMAC() MACAddr {
	mac := OnesMAC
	// TODO
	return mac
}

func (d *DiscoveryTinkerbellV1) Hostname() (string, error) {
	return d.Instance().Hostname, nil // temp
}

func (d *DiscoveryTinkerbellV1) SetMac(mac net.HardwareAddr) {
	d.mac = mac
}

func (h HardwareTinkerbellV1) HardwareAllowPXE(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).Netboot.AllowPXE
}

func (h HardwareTinkerbellV1) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).Netboot.AllowWorkflow
}

func (h HardwareTinkerbellV1) HardwareArch(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).DHCP.Arch
}

func (h HardwareTinkerbellV1) HardwareBondingMode() BondingMode {
	return h.Metadata.BondingMode
}

func (h HardwareTinkerbellV1) HardwareFacilityCode() string {
	return h.Metadata.Facility.FacilityCode
}

func (h HardwareTinkerbellV1) HardwareID() string {
	return h.ID
}

func (h HardwareTinkerbellV1) HardwareIPs() []IP {
	var hips []IP
	// TODO
	return hips
}

//func (h HardwareTinkerbellV1) HardwareIPMI() net.IP {
//	return h.DHCP.IP // is this correct?
//}

func (h HardwareTinkerbellV1) HardwareManufacturer() string {
	return h.Metadata.Manufacturer.Slug
}

func (h HardwareTinkerbellV1) HardwarePlanSlug() string {
	return h.Metadata.Facility.PlanSlug
}

func (h HardwareTinkerbellV1) HardwarePlanVersionSlug() string {
	return h.Metadata.Facility.PlanVersionSlug
}

func (h HardwareTinkerbellV1) HardwareState() HardwareState {
	return h.Metadata.State
}

// dummy method for backward compatibility
func (h HardwareTinkerbellV1) HardwareServicesVersion() string {
	return ""
}

func (h HardwareTinkerbellV1) HardwareUEFI(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).DHCP.UEFI
}

func (h HardwareTinkerbellV1) Interfaces() []Port {
	// TODO: to be updated
	var ports []Port
	return ports
}

func (h HardwareTinkerbellV1) OsieBaseURL(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.BaseURL
}

func (h HardwareTinkerbellV1) KernelPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.Kernel
}

func (h HardwareTinkerbellV1) InitrdPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.Osie.Initrd
}
