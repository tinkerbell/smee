package packet

import (
	"net"
	"time"
)

// models_standalone.go contains a standalone backend for boots so it can run without
// a packet or tinkerbell backend. Instead of using a scalable backend, hardware data
// is loaded from a yaml file and stored in a list in memory.

// DiscoveryStandalone implements the Discovery interface for standalone operation
type DiscoverStandalone struct {
	*HardwareStandalone
	mac net.HardwareAddr
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
	return &Instance{}
}
func (ds DiscoverStandalone) MAC() net.HardwareAddr {
	return net.HardwareAddr{}
}
func (ds DiscoverStandalone) Mode() string {
	return "testing"
}
func (ds DiscoverStandalone) GetIP(addr net.HardwareAddr) IP {
	return IP{}
}
func (ds DiscoverStandalone) GetMAC(ip net.IP) net.HardwareAddr {
	return net.HardwareAddr{}
}
func (ds DiscoverStandalone) DnsServers(mac net.HardwareAddr) []net.IP {
	return []net.IP{}
}
func (ds DiscoverStandalone) LeaseTime(mac net.HardwareAddr) time.Duration {
	return time.Hour
}
func (ds DiscoverStandalone) Hostname() (string, error) {
	return "", nil
}
func (ds DiscoverStandalone) Hardware() Hardware {
	return HardwareStandalone{}
}

func (ds DiscoverStandalone) SetMAC(mac net.HardwareAddr) {}

func (hs HardwareStandalone) HardwareAllowPXE(mac net.HardwareAddr) bool {
	return true
}
func (hs HardwareStandalone) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return true
}
func (hs HardwareStandalone) HardwareArch(mac net.HardwareAddr) string {
	return ""
}
func (hs HardwareStandalone) HardwareBondingMode() BondingMode {
	return 0
}
func (hs HardwareStandalone) HardwareFacilityCode() string {
	return ""
}
func (hs HardwareStandalone) HardwareID() HardwareID {
	return ""
}
func (hs HardwareStandalone) HardwareIPs() []IP {
	return []IP{}
}
func (hs HardwareStandalone) Interfaces() []Port {
	return []Port{}
}
func (hs HardwareStandalone) HardwareManufacturer() string {
	return ""
}
func (hs HardwareStandalone) HardwareProvisioner() string {
	return ""
}
func (hs HardwareStandalone) HardwarePlanSlug() string {
	return ""
}
func (hs HardwareStandalone) HardwarePlanVersionSlug() string {
	return ""
}
func (hs HardwareStandalone) HardwareState() HardwareState {
	return ""
}
func (hs HardwareStandalone) HardwareOSIEVersion() string {
	return ""
}
func (hs HardwareStandalone) HardwareUEFI(mac net.HardwareAddr) bool {
	return true
}
func (hs HardwareStandalone) OSIEBaseURL(mac net.HardwareAddr) string {
	return ""
}
func (hs HardwareStandalone) KernelPath(mac net.HardwareAddr) string {
	return ""
}
func (hs HardwareStandalone) InitrdPath(mac net.HardwareAddr) string {
	return ""
}
func (hs HardwareStandalone) OperatingSystem() *OperatingSystem {
	return &OperatingSystem{}
}
