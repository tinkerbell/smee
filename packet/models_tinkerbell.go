package packet

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/conf"
)

func (i InterfaceTinkerbell) Name() string {
	return i.IfaceName
}

func (d DiscoveryTinkerbell) LeaseTime() time.Duration {
	return d.DHCP.LeaseTime
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

func (d DiscoveryTinkerbell) Ip(mac net.HardwareAddr) IP {
	// TODO
	return IP{}
}

func (dt *DiscoveryTinkerbell) PrimaryDataMAC() MACAddr {
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

func (h HardwareTinkerbell) HardwareAllowPXE() bool {
	return h.Netboot.AllowPXE
}

func (h HardwareTinkerbell) HardwareAllowWorkflow() bool {
	return h.Netboot.AllowWorkflow
}

func (h HardwareTinkerbell) HardwareArch() string {
	return h.DHCP.Arch
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

func (h HardwareTinkerbell) HardwareServicesVersion() Osie {
	return h.Netboot.Bootstrapper
}

func (h HardwareTinkerbell) HardwareUEFI() bool {
	return h.DHCP.UEFI
}

func (h *HardwareTinkerbell) Interfaces() []Port {
	var ports []Port
	return ports
}
