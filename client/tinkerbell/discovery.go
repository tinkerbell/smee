package tinkerbell

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/conf"
)

//go:generate mockgen -destination mock_workflow/workflow_mock.go github.com/tinkerbell/tink/protos/workflow WorkflowServiceClient
//go:generate mockgen -destination mock_hardware/hardware_mock.go github.com/tinkerbell/tink/protos/hardware HardwareServiceClient

// models_tinkerbell.go contains the interface methods specific to DiscoveryTinkerbell and HardwareTinkerbell structs

// DiscoveryTinkerbellV1 presents the structure for tinkerbell's new data model, version 1.
type DiscoveryTinkerbellV1 struct {
	*HardwareTinkerbellV1
	mac net.HardwareAddr
}

// HardwareTinkerbellV1 represents the new hardware data model for tinkerbell, version 1.
type HardwareTinkerbellV1 struct {
	ID       string          `json:"id"`
	Network  client.Network  `json:"network"`
	Metadata client.Metadata `json:"metadata"`
}

func (d DiscoveryTinkerbellV1) LeaseTime(mac net.HardwareAddr) time.Duration {
	leaseTime := d.Network.InterfaceByMac(mac).DHCP.LeaseTime
	if leaseTime == 0 {
		return conf.DHCPLeaseTime
	}
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", leaseTime))

	return duration
}

func (d DiscoveryTinkerbellV1) Hardware() client.Hardware {
	return d.HardwareTinkerbellV1
}

func (d DiscoveryTinkerbellV1) DNSServers(mac net.HardwareAddr) []net.IP {
	dnsServers := d.Network.InterfaceByMac(mac).DHCP.NameServers
	if len(dnsServers) == 0 {
		return conf.DNSServers
	}

	return conf.ParseIPv4s(strings.Join(dnsServers, ","))
}

func (d DiscoveryTinkerbellV1) Instance() *client.Instance {
	return d.Metadata.Instance
}

func (d DiscoveryTinkerbellV1) MAC() net.HardwareAddr {
	return d.mac
}

func (d DiscoveryTinkerbellV1) Mode() string {
	return "hardware"
}

func (d DiscoveryTinkerbellV1) GetIP(mac net.HardwareAddr) client.IP {
	return d.Network.InterfaceByMac(mac).DHCP.IP
}

func (d DiscoveryTinkerbellV1) GetMAC(ip net.IP) net.HardwareAddr {
	return d.Network.InterfaceByIP(ip).DHCP.MAC.HardwareAddr()
}

func (d *DiscoveryTinkerbellV1) PrimaryDataMAC() client.MACAddr {
	mac := client.OnesMAC
	// TODO
	return mac
}

func (d DiscoveryTinkerbellV1) Hostname() (string, error) {
	if d.Instance() == nil {
		return "", nil
	}

	return d.Instance().Hostname, nil
}

func (d *DiscoveryTinkerbellV1) SetMAC(mac net.HardwareAddr) {
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

func (h HardwareTinkerbellV1) HardwareBondingMode() client.BondingMode {
	return h.Metadata.BondingMode
}

func (h HardwareTinkerbellV1) HardwareFacilityCode() string {
	return h.Metadata.Facility.FacilityCode
}

func (h HardwareTinkerbellV1) HardwareID() client.HardwareID {
	return client.HardwareID(h.ID)
}

func (h HardwareTinkerbellV1) HardwareIPs() []client.IP {
	// TODO
	return []client.IP{}
}

// func (h HardwareTinkerbellV1) HardwareIPMI() net.IP {
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

func (h HardwareTinkerbellV1) HardwareState() client.HardwareState {
	return h.Metadata.State
}

// dummy method for backward compatibility.
func (h HardwareTinkerbellV1) HardwareOSIEVersion() string {
	return ""
}

func (h HardwareTinkerbellV1) HardwareUEFI(mac net.HardwareAddr) bool {
	return h.Network.InterfaceByMac(mac).DHCP.UEFI
}

func (h HardwareTinkerbellV1) Interfaces() []client.Port {
	// TODO: to be updated
	var ports []client.Port

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

func (h *HardwareTinkerbellV1) OperatingSystem() *client.OperatingSystem {
	i := h.instance()
	if i.OS == nil {
		i.OS = &client.OperatingSystem{}
	}

	return i.OS
}

func (h *HardwareTinkerbellV1) instance() *client.Instance {
	if h.Metadata.Instance == nil {
		h.Metadata.Instance = &client.Instance{}
	}

	return h.Metadata.Instance
}

// GetTraceparent always returns empty string.
// TODO(@tobert, 2021-11-30): implement this.
func (h HardwareTinkerbellV1) GetTraceparent() string {
	return ""
}
