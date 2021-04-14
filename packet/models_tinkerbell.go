package packet

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/tinkerbell/boots/conf"
)

//go:generate mockgen -destination mock_workflow/workflow_mock.go github.com/tinkerbell/tink/protos/workflow WorkflowServiceClient
//go:generate mockgen -destination mock_hardware/hardware_mock.go github.com/tinkerbell/tink/protos/hardware HardwareServiceClient

// models_tinkerbell.go contains the interface methods specific to DiscoveryTinkerbell and HardwareTinkerbell structs

// DiscoveryTinkerbellV1 presents the structure for tinkerbell's new data model, version 1
type DiscoveryTinkerbellV1 struct {
	*HardwareTinkerbellV1
	mac net.HardwareAddr
}

// HardwareTinkerbellV1 represents the new hardware data model for tinkerbell, version 1
type HardwareTinkerbellV1 struct {
	ID       string   `json:"id"`
	Network  Network  `json:"network"`
	Metadata Metadata `json:"metadata"`
}

func (i InterfaceTinkerbell) Name() string {
	return i.DHCP.IfaceName
}

func (d DiscoveryTinkerbellV1) LeaseTime(mac net.HardwareAddr) time.Duration {
	leaseTime := d.Network.InterfaceByMac(mac).DHCP.LeaseTime
	if leaseTime == 0 {
		return conf.DHCPLeaseTime
	}
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", leaseTime))
	return duration
}

func (d DiscoveryTinkerbellV1) Hardware() Hardware {
	var h Hardware = d.HardwareTinkerbellV1
	return h
}

func (d DiscoveryTinkerbellV1) DnsServers(mac net.HardwareAddr) []net.IP {
	dnsServers := d.Network.InterfaceByMac(mac).DHCP.NameServers
	if len(dnsServers) == 0 {
		return conf.DNSServers
	}
	return conf.ParseIPv4s(strings.Join(dnsServers, ","))
}

func (d DiscoveryTinkerbellV1) Instance() *Instance {
	return d.Metadata.Instance
}

func (d DiscoveryTinkerbellV1) MAC() net.HardwareAddr {
	return d.mac
}

func (d DiscoveryTinkerbellV1) Mode() string {
	return "hardware"
}

func (d DiscoveryTinkerbellV1) GetIP(mac net.HardwareAddr) IP {
	//if i := d.Network.Interface(mac); i.DHCP.IP.Address != nil {
	//	return i.DHCP.IP
	//}
	return d.Network.InterfaceByMac(mac).DHCP.IP
}

func (d DiscoveryTinkerbellV1) GetMAC(ip net.IP) net.HardwareAddr {
	return d.Network.InterfaceByIp(ip).DHCP.MAC.HardwareAddr()
}

// InterfacesByMac returns the NetworkInterface that contains the matching mac address
// returns an empty NetworkInterface if not found
func (n Network) InterfaceByMac(mac net.HardwareAddr) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.MAC.String() == mac.String() {
			return i
		}
	}
	return NetworkInterface{}
}

// InterfacesByIp returns the NetworkInterface that contains the matching ip address
// returns an empty NetworkInterface if not found
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

func (d DiscoveryTinkerbellV1) Hostname() (string, error) {
	if d.Instance() == nil {
		return "", nil
	}
	return d.Instance().Hostname, nil
}

func (d DiscoveryTinkerbellV1) SetMAC(mac net.HardwareAddr) {
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

func (h HardwareTinkerbellV1) HardwareID() HardwareID {
	return HardwareID(h.ID)
}

func (h HardwareTinkerbellV1) HardwareIPs() []IP {
	var hips []IP
	// TODO
	return hips
}

//func (h HardwareTinkerbellV1) HardwareIPMI() net.IP {
//	return h.DHCP.IP // is this correct?
//}

func (h HardwareTinkerbellV1) HardwareProvisioner() string {
	return h.Metadata.ProvisionerEngine
}

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
func (h HardwareTinkerbellV1) HardwareOSIEVersion() string {
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

func (h HardwareTinkerbellV1) OSIEBaseURL(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.OSIE.BaseURL
}

func (h HardwareTinkerbellV1) KernelPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.OSIE.Kernel
}

func (h HardwareTinkerbellV1) InitrdPath(mac net.HardwareAddr) string {
	return h.Network.InterfaceByMac(mac).Netboot.OSIE.Initrd
}

func (h *HardwareTinkerbellV1) OperatingSystem() *OperatingSystem {
	i := h.instance()
	if i.OS == nil {
		i.OS = &OperatingSystem{}
	}
	return i.OS
}

func (h *HardwareTinkerbellV1) instance() *Instance {
	if h.Metadata.Instance == nil {
		h.Metadata.Instance = &Instance{}
	}
	return h.Metadata.Instance
}
