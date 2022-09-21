package standalone

import (
	"net"

	"github.com/tinkerbell/boots/client"
)

// HardwareStandalone implements the Hardware interface for standalone operation.
type HardwareStandalone struct {
	ID          string          `json:"id"`
	Network     client.Network  `json:"network"`
	Metadata    client.Metadata `json:"metadata"`
	Traceparent string          `json:"traceparent"`
}

func (hs *HardwareStandalone) HardwareAllowPXE(net.HardwareAddr) bool {
	return hs.getPrimaryInterface().Netboot.AllowPXE
}

func (hs *HardwareStandalone) HardwareAllowWorkflow(net.HardwareAddr) bool {
	return hs.getPrimaryInterface().Netboot.AllowWorkflow
}

func (hs *HardwareStandalone) HardwareArch(net.HardwareAddr) string {
	return hs.getPrimaryInterface().DHCP.Arch
}

func (hs *HardwareStandalone) HardwareBondingMode() client.BondingMode {
	return hs.Metadata.BondingMode
}

func (hs *HardwareStandalone) HardwareFacilityCode() string {
	return hs.Metadata.Facility.FacilityCode
}

func (hs *HardwareStandalone) HardwareID() client.HardwareID {
	return client.HardwareID(hs.ID)
}

func (hs *HardwareStandalone) HardwareIPs() []client.IP {
	out := make([]client.IP, len(hs.Network.Interfaces))
	for i, v := range hs.Network.Interfaces {
		out[i] = v.DHCP.IP
	}

	return out
}

func (hs *HardwareStandalone) Interfaces() []client.Port {
	return []client.Port{} // stubbed out in tink too
}

func (hs *HardwareStandalone) HardwareManufacturer() string {
	return hs.Metadata.Manufacturer.Slug
}

func (hs *HardwareStandalone) HardwareProvisioner() string {
	return hs.Metadata.ProvisionerEngine
}

func (hs *HardwareStandalone) HardwarePlanSlug() string {
	return hs.Metadata.Facility.PlanSlug
}

func (hs *HardwareStandalone) HardwarePlanVersionSlug() string {
	return hs.Metadata.Facility.PlanVersionSlug
}

func (hs *HardwareStandalone) HardwareState() client.HardwareState {
	return hs.Metadata.State
}

func (hs *HardwareStandalone) HardwareOSIEVersion() string {
	return "" // stubbed out in tink too
}

func (hs *HardwareStandalone) HardwareUEFI(net.HardwareAddr) bool {
	return hs.getPrimaryInterface().DHCP.UEFI
}

func (hs *HardwareStandalone) OSIEBaseURL(net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.BaseURL
}

func (hs *HardwareStandalone) KernelPath(net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.Kernel
}

func (hs *HardwareStandalone) InitrdPath(net.HardwareAddr) string {
	return hs.getPrimaryInterface().Netboot.OSIE.Initrd
}

func (hs *HardwareStandalone) OperatingSystem() *client.OperatingSystem {
	return hs.Metadata.Instance.OS
}

// getPrimaryInterface returns the first interface in the list of interfaces
// and returns an empty interface with zeroed MAC & empty IP if that list is
// empty. Other models have more sophisticated interface selection but this
// model is mostly intended for test so this might behave in unexpected ways
// with more complex configurations and need more logic added.
func (hs *HardwareStandalone) getPrimaryInterface() client.NetworkInterface {
	if len(hs.Network.Interfaces) >= 1 {
		return hs.Network.Interfaces[0]
	}

	return hs.emptyInterface()
}

func (hs *HardwareStandalone) emptyInterface() client.NetworkInterface {
	return client.NetworkInterface{
		DHCP: client.DHCP{
			MAC:         &client.MACAddr{},
			IP:          client.IP{},
			NameServers: []string{},
			TimeServers: []string{},
		},
		Netboot: client.Netboot{},
	}
}

// GetTraceparent returns the traceparent from the config.
func (hs *HardwareStandalone) GetTraceparent() string {
	return hs.Traceparent
}

// GetVLANID returns the VLAN ID from the config.
func (hs *HardwareStandalone) GetVLANID(net.HardwareAddr) string {
	return ""
}
