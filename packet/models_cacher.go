package packet

import (
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
)

// models_cacher.go contains the interface methods specific to DiscoveryCacher and HardwareCacher structs

//go:generate mockgen -destination mock_cacher/cacher_mock.go github.com/packethost/cacher/protos/cacher CacherClient

// DiscoveryCacher presents the structure for old data model
type DiscoveryCacher struct {
	*HardwareCacher
	mac net.HardwareAddr
}

// HardwareCacher represents the old hardware data model for backward compatibility
type HardwareCacher struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	State HardwareState `json:"state"`

	BondingMode       BondingMode     `json:"bonding_mode"`
	NetworkPorts      []Port          `json:"network_ports"`
	Manufacturer      Manufacturer    `json:"manufacturer"`
	PlanSlug          string          `json:"plan_slug"`
	PlanVersionSlug   string          `json:"plan_version_slug"`
	Arch              string          `json:"arch"`
	FacilityCode      string          `json:"facility_code"`
	IPMI              IP              `json:"management"`
	IPs               []IP            `json:"ip_addresses"`
	PreinstallOS      OperatingSystem `json:"preinstalled_operating_system_version"`
	PrivateSubnets    []string        `json:"private_subnets,omitempty"`
	UEFI              bool            `json:"efi_boot"`
	AllowPXE          bool            `json:"allow_pxe"`
	AllowWorkflow     bool            `json:"allow_workflow"`
	ServicesVersion   ServicesVersion `json:"services"`
	Instance          *Instance       `json:"instance"`
	ProvisionerEngine string          `json:"provisioner_engine"`
}

func (d DiscoveryCacher) Hardware() Hardware {
	var h Hardware = d.HardwareCacher

	return h
}

func (d DiscoveryCacher) DnsServers(mac net.HardwareAddr) []net.IP {
	return conf.DNSServers
}

func (d DiscoveryCacher) Instance() *Instance {
	return d.HardwareCacher.Instance
}

func (d DiscoveryCacher) LeaseTime(mac net.HardwareAddr) time.Duration {
	return conf.DHCPLeaseTime
}

func (d DiscoveryCacher) MAC() net.HardwareAddr {
	if d.mac == nil {
		mac := d.PrimaryDataMAC()

		return mac.HardwareAddr()
	}

	return d.mac
}

func (d DiscoveryCacher) MacType(mac string) string {
	for _, port := range d.NetworkPorts {
		if port.MAC().String() == mac {
			return string(port.Type)
		}
	}

	return "NOTFOUND"
}

func (d DiscoveryCacher) MacIsType(mac string, portType string) bool {
	for _, port := range d.NetworkPorts {
		if port.MAC().String() != mac {
			continue
		}

		return string(port.Type) == portType
	}

	return false
}

// Mode returns whether the mac belongs to the instance, hardware, bmc, discovered, or unknown
func (d DiscoveryCacher) Mode() string {
	mac := d.mac
	if d.InstanceIP(mac.String()) != nil {
		return "instance"
	}
	if d.ManagementIP(mac.String()) != nil {
		return "management"
	}
	if d.HardwareIP(mac.String()) != nil {
		return "hardware"
	}
	if d.DiscoveredIP(mac.String()) != nil {
		return "discovered"
	}

	return ""
}

// NetConfig returns the network configuration that corresponds to the interface whose MAC address is mac.
func (d DiscoveryCacher) GetIP(mac net.HardwareAddr) IP {
	ip := d.InstanceIP(mac.String())
	if ip != nil {
		return *ip
	}
	ip = d.ManagementIP(mac.String())
	if ip != nil {
		return *ip
	}
	ip = d.HardwareIP(mac.String())
	if ip != nil {
		return *ip
	}
	ip = d.DiscoveredIP(mac.String())
	if ip != nil {
		return *ip
	}

	return IP{}
}

// dummy method for tink data model transition
func (d DiscoveryCacher) GetMAC(ip net.IP) net.HardwareAddr {
	return d.PrimaryDataMAC().HardwareAddr()
}

// InstanceIP returns the IP configuration that should be Offered to the instance if there is one; if it's prov/deprov'ing, it's the hardware IP
func (d DiscoveryCacher) InstanceIP(mac string) *IP {
	if d.Instance() == nil || d.Instance().ID == "" || !d.MacIsType(mac, "data") || d.PrimaryDataMAC().HardwareAddr().String() != mac {
		return nil
	}
	if ip := d.Instance().FindIP(managementPublicIPv4IP); ip != nil {
		return ip
	}
	if ip := d.Instance().FindIP(managementPrivateIPv4IP); ip != nil {
		return ip
	}
	if d.Instance().State == "provisioning" || d.Instance().State == "deprovisioning" {
		ip := d.hardwareIP()
		if ip != nil {
			return ip
		}
	}

	return nil
}

// HardwareIP returns the IP configuration that should be offered to the hardware if there is no instance
func (d DiscoveryCacher) HardwareIP(mac string) *IP {
	if !d.MacIsType(mac, "data") {
		return nil
	}
	if d.PrimaryDataMAC().HardwareAddr().String() != mac {
		return nil
	}

	return d.hardwareIP()
}

// hardwareIP returns the IP configuration that should be offered to the hardware (not exported)
func (d DiscoveryCacher) hardwareIP() *IP {
	h := d.Hardware()
	for _, ip := range h.HardwareIPs() {
		if ip.Family != 4 {
			continue
		}
		if ip.Public {
			continue
		}

		return &ip
	}

	return nil
}

// ManagementIP returns the IP configuration that should be Offered to the BMC, if the MAC is a BMC MAC
func (d DiscoveryCacher) ManagementIP(mac string) *IP {
	if d.MacIsType(mac, "ipmi") && d.Name != "" {
		return &d.IPMI
	}

	return nil
}

// DiscoveredIP returns the IP configuration that should be offered to a newly discovered BMC, if the MAC is a BMC MAC
func (d DiscoveryCacher) DiscoveredIP(mac string) *IP {
	if d.MacIsType(mac, "ipmi") && d.Name == "" {
		return &d.IPMI
	}

	return nil
}

// PrimaryDataMAC returns the mac associated with eth0, or as a fallback the lowest numbered non-bmc MAC address
func (d DiscoveryCacher) PrimaryDataMAC() MACAddr {
	mac := OnesMAC
	for _, port := range d.NetworkPorts {
		if port.Type != "data" {
			continue
		}
		if port.Name == "eth0" {
			mac = *port.Data.MAC

			break
		}
		if port.MAC().String() < mac.String() {
			mac = *port.Data.MAC
		}
	}

	if mac.IsOnes() {
		return ZeroMAC
	}

	return mac
}

// ManagementMAC returns the mac address of the BMC interface
func (d DiscoveryCacher) ManagementMAC() MACAddr {
	for _, port := range d.NetworkPorts {
		if port.Type == "ipmi" {
			return *port.Data.MAC
		}
	}

	return ZeroMAC
}

func (d DiscoveryCacher) Hostname() (string, error) {
	var hostname string

	mode := d.Mode()
	switch mode {
	case "discovered", "management":
		hostname = d.Name
	case "instance":
		hostname = d.Instance().Hostname
		switch d.State {
		case "deprovisioning":
			hostname = d.Name
		case "provisioning":
		}
	case "hardware":
		hostname = d.Name
	default:
		return "", errors.Errorf("unknown mode: %s", mode)
	}

	return hostname, nil
}

func (d *DiscoveryCacher) SetMAC(mac net.HardwareAddr) {
	d.mac = mac
}

func (h *HardwareCacher) Management() (address, netmask, gateway net.IP) {
	ip := h.IPMI

	return ip.Address, ip.Netmask, ip.Gateway
}

func (h HardwareCacher) Interfaces() []Port {
	ports := make([]Port, 0, len(h.NetworkPorts)-1)
	for _, p := range h.NetworkPorts {
		if p.Type == "ipmi" {
			continue
		}
		ports = append(ports, p)
	}
	if len(ports) == 0 {
		return nil
	}

	return ports
}

func (i InterfaceCacher) Name() string {
	return i.Port.Name
}

func (h HardwareCacher) HardwareAllowPXE(mac net.HardwareAddr) bool {
	return h.AllowPXE
}

func (h HardwareCacher) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return h.AllowWorkflow
}

func (h HardwareCacher) HardwareArch(mac net.HardwareAddr) string {
	return h.Arch
}

func (h HardwareCacher) HardwareBondingMode() BondingMode {
	return h.BondingMode
}

func (h HardwareCacher) HardwareFacilityCode() string {
	return h.FacilityCode
}

func (h HardwareCacher) HardwareID() HardwareID {
	return HardwareID(h.ID)
}

func (h HardwareCacher) HardwareIPs() []IP {
	return h.IPs
}

func (h HardwareCacher) HardwareIPMI() IP {
	return h.IPMI
}

func (h HardwareCacher) HardwareManufacturer() string {
	return h.Manufacturer.Slug
}

func (h HardwareCacher) HardwareProvisioner() string {
	return h.ProvisionerEngine
}

func (h HardwareCacher) HardwarePlanSlug() string {
	return h.PlanSlug
}

func (h HardwareCacher) HardwarePlanVersionSlug() string {
	return h.PlanVersionSlug
}

func (h HardwareCacher) HardwareOSIEVersion() string {
	return h.ServicesVersion.OSIE
}

func (h HardwareCacher) HardwareState() HardwareState {
	return h.State
}

func (h HardwareCacher) HardwareUEFI(mac net.HardwareAddr) bool {
	return h.UEFI
}

// dummy method for tink data model transition
func (h HardwareCacher) OSIEBaseURL(mac net.HardwareAddr) string {
	return ""
}

// dummy method for tink data model transition
func (h HardwareCacher) KernelPath(mac net.HardwareAddr) string {
	return ""
}

// dummy method for tink data model transition
func (h HardwareCacher) InitrdPath(mac net.HardwareAddr) string {
	return ""
}

func (h *HardwareCacher) OperatingSystem() *OperatingSystem {
	i := h.instance()
	if i.OSV == (*OperatingSystem)(nil) {
		i.OSV = &OperatingSystem{}
	}

	return i.OSV
}

func (h *HardwareCacher) instance() *Instance {
	if h.Instance == nil {
		h.Instance = &Instance{}
	}

	return h.Instance
}
