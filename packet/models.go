package packet

import (
	"encoding/json"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
)

// models.go contains the Hardware structures matching the data models defined by tink and cacher

// BondingMode is the hardware bonding mode
type BondingMode int

// Discovery interface is the base for cacher and tinkerbell hardware discovery
type Discovery interface {
	Instance() *Instance
	MAC() net.HardwareAddr
	Mode() string
	GetIP(addr net.HardwareAddr) IP
	GetMAC(ip net.IP) net.HardwareAddr
	DnsServers(mac net.HardwareAddr) []net.IP
	LeaseTime(mac net.HardwareAddr) time.Duration
	Hostname() (string, error)
	Hardware() Hardware
	SetMAC(mac net.HardwareAddr)
}

// DiscoveryCacher presents the structure for old data model
type DiscoveryCacher struct {
	*HardwareCacher
	mac net.HardwareAddr
}

// DiscoveryTinkerbellV1 presents the structure for tinkerbell's new data model, version 1
type DiscoveryTinkerbellV1 struct {
	*HardwareTinkerbellV1
	mac net.HardwareAddr
}

// Interface is the base for cacher and tinkerbell hardware (network) interface
type Interface interface {
}

type InterfaceCacher struct {
	*Port
}

type InterfaceTinkerbell struct {
	*NetworkInterface
}

// Hardware interface holds primary hardware methods
type Hardware interface {
	HardwareAllowPXE(mac net.HardwareAddr) bool
	HardwareAllowWorkflow(mac net.HardwareAddr) bool
	HardwareArch(mac net.HardwareAddr) string
	HardwareBondingMode() BondingMode
	HardwareFacilityCode() string
	HardwareID() string
	HardwareIPs() []IP
	Interfaces() []Port // TODO: to be updated
	HardwareManufacturer() string
	HardwarePlanSlug() string
	HardwarePlanVersionSlug() string
	HardwareState() HardwareState
	HardwareServicesVersion() string
	HardwareUEFI(mac net.HardwareAddr) bool
	OsieBaseURL(mac net.HardwareAddr) string
	KernelPath(mac net.HardwareAddr) string
	InitrdPath(mac net.HardwareAddr) string
}

// HardwareCacher represents the old hardware data model for backward compatibility
type HardwareCacher struct {
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
	AllowWorkflow   bool            `json:"allow_workflow"`
	ServicesVersion ServicesVersion `json:"services"`
	Instance        *Instance       `json:"instance"`
}

// HardwareTinkerbellV1 represents the new hardware data model for tinkerbell, version 1
type HardwareTinkerbellV1 struct {
	ID       string   `json:"id"`
	Network  Network  `json:"network"`
	Metadata Metadata `json:"metadata"`
}

// NewDiscovery instantiates a Discovery struct from the json argument
func NewDiscovery(b []byte) (Discovery, error) {
	if string(b) == "" || string(b) == "{}" {
		return nil, errors.New("empty response from db")
	}

	dataModelVersion := os.Getenv("DATA_MODEL_VERSION")
	switch dataModelVersion {
	case "1":
		d := &DiscoveryTinkerbellV1{}
		err := json.Unmarshal(b, &d)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal json for discovery")
		}
		return d, nil
	case "":
		d := &DiscoveryCacher{}
		err := json.Unmarshal(b, &d)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal json for discovery")
		}
		return d, nil
	default:
		return nil, errors.New("unknown DATA_MODEL_VERSION")
	}
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

	Tags []string `json:"tags,omitempty"`
	// Project
	SSHKeys []string `json:"ssh_keys,omitempty"`
	// CustomData
	NetworkReady bool `json:"network_ready,omitempty"`
}

// Device Full device result from /devices endpoint
type Device struct {
	ID string `json:"id"`
}

// FindIP returns IP for an instance, nil otherwise
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

// InstanceState represents the state of an instance (e.g. active)
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

// HardwareState is the hardware state (e.g. provisioning)
type HardwareState string

// IP represents IP address for a hardware
type IP struct {
	Address    net.IP `json:"address"`
	Netmask    net.IP `json:"netmask"`
	Gateway    net.IP `json:"gateway"`
	Family     int    `json:"address_family"`
	Public     bool   `json:"public"`
	Management bool   `json:"management"`
}

// OperatingSystem holds details for the operating system
type OperatingSystem struct {
	Slug     string `json:"slug"`
	Distro   string `json:"distro"`
	Version  string `json:"version"`
	ImageTag string `json:"image_tag"`
	OsSlug   string `json:"os_slug"`
}

// Port represents a network port
type Port struct {
	ID   string   `json:"id"`
	Type PortType `json:"type"`
	Name string   `json:"name"`
	Data struct {
		MAC  *MACAddr `json:"mac"`
		Bond string   `json:"bond"`
	} `json:"data"`
}

// MAC returns the physical hardware address, nil otherwise
func (p *Port) MAC() net.HardwareAddr {
	if p.Data.MAC != nil && *p.Data.MAC != ZeroMAC {
		return p.Data.MAC.HardwareAddr()
	}
	return nil
}

// PortType is type for a network port
type PortType string

// Manufacturer holds data for hardware manufacturer
type Manufacturer struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type NetworkInterface struct {
	DHCP    DHCP    `json:"dhcp,omitempty"`
	Netboot Netboot `json:"netboot,omitempty"`
}

// DHCP holds details for DHCP connection
type DHCP struct {
	MAC         *MACAddr `json:"mac"`
	IP          IP       `json:"ip"`
	Hostname    string   `json:"hostname"`
	LeaseTime   int      `json:"lease_time"`
	NameServers []string `json:"name_servers"`
	TimeServers []string `json:"time_servers"`
	Arch        string   `json:"arch"`
	UEFI        bool     `json:"uefi"`
	IfaceName   string   `json:"iface_name"` // to be removed?
}

// Netboot holds details for a hardware to boot over network
type Netboot struct {
	AllowPXE      bool `json:"allow_pxe"`      // to be removed?
	AllowWorkflow bool `json:"allow_workflow"` // to be removed?
	IPXE          struct {
		URL      string `json:"url"`
		Contents string `json:"contents"`
	} `json:"ipxe"`
	Osie Osie `json:"osie"`
}

// Bootstrapper is the bootstrapper to be used during netboot
type Osie struct {
	BaseURL string `json:"base_url"`
	Kernel  string `json:"kernel"`
	Initrd  string `json:"initrd"`
}

// Network holds hardware network details
type Network struct {
	Interfaces []NetworkInterface `json:"interfaces,omitempty"`
	//Default    NetworkInterface   `json:"default,omitempty"`
}

// Metadata holds the hardware metadata
type Metadata struct {
	State        HardwareState `json:"state"`
	BondingMode  BondingMode   `json:"bonding_mode"`
	Manufacturer Manufacturer  `json:"manufacturer"`
	Instance     *Instance     `json:"instance"`
	Custom       struct {
		PreinstalledOS OperatingSystem `json:"preinstalled_operating_system_version"`
		PrivateSubnets []string        `json:"private_subnets"`
	} `json:"custom"`
	Facility Facility `json:"facility"`
}

// Facility represents the facilty in use
type Facility struct {
	PlanSlug        string `json:"plan_slug"`
	PlanVersionSlug string `json:"plan_version_slug"`
	FacilityCode    string `json:"facility_code"`
}
