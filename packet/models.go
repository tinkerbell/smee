package packet

import (
	"bytes"
	"encoding/json"
	"net"
	"sort"

	"github.com/pkg/errors"
)

type BondingMode int

type Discovery struct {
	*Hardware
}

func (d *Discovery) macType(mac string) string {
	for _, port := range d.NetworkPorts {
		if port.MAC().String() == mac {
			return string(port.Type)
		}
	}
	return "NOTFOUND"
}

func (d *Discovery) macIsType(mac string, portType string) bool {
	for _, port := range d.NetworkPorts {
		if port.MAC().String() != mac {
			continue
		}
		return string(port.Type) == portType
	}
	return false
}

// New instantiates a Discovery struct from the json argument
func NewDiscovery(j string) (*Discovery, error) {
	var res Discovery
	err := json.Unmarshal([]byte(j), &res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}
	return &res, err
}

// Mode returns whether the mac belongs to the instance, hardware, bmc, discovered, or unknown
func (d Discovery) Mode(mac net.HardwareAddr) string {
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
func (d Discovery) NetConfig(mac net.HardwareAddr) IP {
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

// InstanceIP returns the IP configuration that should be Offered to the instance if there is one; if it's prov/deprov'ing, it's the hardware IP
func (d Discovery) InstanceIP(mac string) *IP {
	if d.Instance == nil || d.Instance.ID == "" || !d.macIsType(mac, "data") || d.PrimaryDataMAC().HardwareAddr().String() != mac {
		return nil
	}
	if ip := d.Instance.FindIP(managementPublicIPv4IP); ip != nil {
		return ip
	}
	if ip := d.Instance.FindIP(managementPrivateIPv4IP); ip != nil {
		return ip
	}
	if d.Instance.State == "provisioning" || d.Instance.State == "deprovisioning" {
		ip := d.hardwareIP()
		if ip != nil {
			return ip
		}
	}
	return nil
}

// HardwareIP returns the IP configuration that should be offered to the hardware if there is no instance
func (d Discovery) HardwareIP(mac string) *IP {
	if !d.macIsType(mac, "data") {
		return nil
	}
	if d.PrimaryDataMAC().HardwareAddr().String() != mac {
		return nil
	}
	return d.hardwareIP()
}

// hardwareIP returns the IP configuration that should be offered to the hardware (not exported)
func (d Discovery) hardwareIP() *IP {
	for _, ip := range d.Hardware.IPs {
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
func (d Discovery) ManagementIP(mac string) *IP {
	if d.macIsType(mac, "ipmi") && d.Name != "" {
		return &d.IPMI
	}
	return nil
}

// DiscoveredIP returns the IP configuration that should be offered to a newly discovered BMC, if the MAC is a BMC MAC
func (d Discovery) DiscoveredIP(mac string) *IP {
	if d.macIsType(mac, "ipmi") && d.Name == "" {
		return &d.IPMI
	}
	return nil
}

// PrimaryDataMAC returns the mac associated with eth0, or as a fallback the lowest numbered non-bmc MAC address
func (d Discovery) PrimaryDataMAC() MACAddr {
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
func (d Discovery) ManagementMAC() MACAddr {
	for _, port := range d.NetworkPorts {
		if port.Type == "ipmi" {
			return *port.Data.MAC
		}
	}
	return ZeroMAC
}

// Instance models the instance data as returned by the API
type Instance struct {
	ID       string        `json:"id"`
	State    InstanceState `json:"state"`
	Hostname string        `json:"hostname"`
	AllowPXE bool          `json:"allow_pxe"`
	Rescue   bool          `json:"rescue"`

	OS            OperatingSystem `json:"operating_system_version"`
	AlwaysPXE     bool            `json:"always_pxe,omitempty"`
	IPXEScriptURL string          `json:"ipxe_script_url,omitempty"`
	IPs           []IP            `json:"ip_addresses"`
	UserData      string          `json:"userdata,omitempty"`

	// Only returned in the first 24 hours
	CryptedRootPassword string `json:"crypted_root_password,omitempty"`
}

// Device Full device result from /devices endpoint
type Device struct {
	ID string `json:"id"`
}

func (i *Instance) FindIP(pred func(IP) bool) *IP {
	for _, ip := range i.IPs {
		if pred(ip) {
			return &ip
		}
	}
	return nil
}

func managementPublicIPv4IP(ip IP) bool {
	return ip.Public && ip.Management && ip.Family == 4
}

func managementPrivateIPv4IP(ip IP) bool {
	return !ip.Public && ip.Management && ip.Family == 4
}

type InstanceState string

type Event struct {
	Type    string `json:"type"`
	Body    string `json:"body,omitempty"`
	Private bool   `json:"private"`
}

type UserEvent struct {
	Code    string `json:"code"`
	State   string `json:"state"`
	Message string `json:"message"`
}

type ServicesVersion struct {
	Osie string `json:"osie"`
}

type Hardware struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	State HardwareState `json:"state"`

	BondingMode     BondingMode     `json:"bonding_mode"`
	NetworkPorts    []Port          `json:"network_ports"`
	Manufacturer    Manufacturer    `json:"manufacturer"`
	PlanSlug        string          `json:"plan_slug"`
	PlanVersionSlug string          `json:"plan_version_slug"`
	Arch            string          `json:"arch"`
	FacilityCode    string          `json:"facility_code"`
	IPMI            IP              `json:"management"`
	IPs             []IP            `json:"ip_addresses"`
	PreinstallOS    OperatingSystem `json:"preinstalled_operating_system_version"`
	PrivateSubnets  []string        `json:"private_subnets,omitempty"`
	UEFI            bool            `json:"efi_boot"`
	AllowPXE        bool            `json:"allow_pxe"`
	ServicesVersion ServicesVersion `json:"services"`

	Instance *Instance `json:"instance"`
}

func (h *Hardware) Management() (address, netmask, gateway net.IP) {
	ip := h.IPMI
	return ip.Address, ip.Netmask, ip.Gateway
}

func (h *Hardware) Interfaces() []Port {
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

type HardwareState string

type IP struct {
	Address    net.IP `json:"address"`
	Netmask    net.IP `json:"netmask"`
	Gateway    net.IP `json:"gateway"`
	Family     int    `json:"address_family"`
	Public     bool   `json:"public"`
	Management bool   `json:"management"`
}

type NetworkPorts struct {
	Main []Port `json:"main"`
	IPMI Port   `json:"ipmi"`
}

func (p *NetworkPorts) addMain(port Port) {
	var (
		mac   = port.MAC()
		ports = p.Main
	)
	n := len(ports)
	i := sort.Search(n, func(i int) bool {
		return bytes.Compare(mac, ports[i].MAC()) < 0
	})
	if i < n {
		ports = append(append(ports[:i], port), ports[i:]...)
	} else {
		ports = append(ports, port)
	}
	p.Main = ports
}

type OperatingSystem struct {
	Slug     string `json:"slug"`
	Distro   string `json:"distro"`
	Version  string `json:"version"`
	ImageTag string `json:"image_tag"`
	OsSlug   string `json:"os_slug"`
}

type Port struct {
	ID   string   `json:"id"`
	Type PortType `json:"type"`
	Name string   `json:"name"`
	Data struct {
		MAC  *MACAddr `json:"mac"`
		Bond string   `json:"bond"`
	} `json:"data"`
}

func (p *Port) MAC() net.HardwareAddr {
	if p.Data.MAC != nil && *p.Data.MAC != ZeroMAC {
		return p.Data.MAC.HardwareAddr()
	}
	return nil
}

type PortType string

type Manufacturer struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}
