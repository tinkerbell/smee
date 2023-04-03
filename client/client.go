package client

import (
	"context"
	"net"
	"time"
)

// HardwareFinder is a type for discovering hardware.
type HardwareFinder interface {
	ByIP(context.Context, net.IP) (Discoverer, error)
	ByMAC(context.Context, net.HardwareAddr, net.IP, string) (Discoverer, error)
}

// Discoverer interface is the base for tinkerbell and standalone hardware discovery.
type Discoverer interface {
	Instance() *Instance
	MAC() net.HardwareAddr
	Mode() string
	GetIP(addr net.HardwareAddr) IP
	GetMAC(ip net.IP) net.HardwareAddr
	DNSServers(mac net.HardwareAddr) []net.IP
	LeaseTime(mac net.HardwareAddr) time.Duration
	Hostname() (string, error)
	Hardware() Hardware
	SetMAC(mac net.HardwareAddr)
}

// Hardware interface holds primary hardware methods.
type Hardware interface {
	HardwareAllowPXE(mac net.HardwareAddr) bool
	HardwareAllowWorkflow(mac net.HardwareAddr) bool
	HardwareArch(mac net.HardwareAddr) string
	HardwareBondingMode() BondingMode
	HardwareFacilityCode() string
	HardwareID() HardwareID
	HardwareIPs() []IP
	Interfaces() []Port // TODO: to be updated
	HardwareManufacturer() string
	HardwareProvisioner() string
	HardwarePlanSlug() string
	HardwarePlanVersionSlug() string
	HardwareState() HardwareState
	HardwareOSIEVersion() string
	HardwareUEFI(mac net.HardwareAddr) bool
	GetVLANID(net.HardwareAddr) string
	OSIEBaseURL(mac net.HardwareAddr) string
	KernelPath(mac net.HardwareAddr) string
	InitrdPath(mac net.HardwareAddr) string
	OperatingSystem() *OperatingSystem
	GetTraceparent() string
	IPXEURL(mac net.HardwareAddr) string
	IPXEScript(mac net.HardwareAddr) string
}

type Component struct {
	Type            string      `json:"type"`
	Name            string      `json:"name"`
	Vendor          string      `json:"vendor"`
	Model           string      `json:"model"`
	Serial          string      `json:"serial"`
	FirmwareVersion string      `json:"firmware_version"`
	Data            interface{} `json:"data"`
}

type ComponentsResponse struct {
	Components []Component `json:"components"`
}

// BondingMode is the hardware bonding mode.
type BondingMode int

type HardwareID string

func (hid HardwareID) String() string {
	return string(hid)
}

// InstanceState represents the state of an instance (e.g. active).
type InstanceState string

// HardwareState is the hardware state (e.g. provisioning).
type HardwareState string
