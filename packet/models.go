package packet

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/files/ignition"
)

const (
	discoveryTypeCacher     = "cacher"
	discoveryTypeTinkerbell = "tinkerbell"
)

type BondingMode int

type Discovery interface {
	Instance() *Instance // temporary
	Mac() net.HardwareAddr
	Mode() string
	Ip(addr net.HardwareAddr) IP
	DnsServers() []net.IP
	LeaseTime() time.Duration
	Hostname() (string, error)
	Hardware() *Hardware
	SetMac(mac net.HardwareAddr)
}

type DiscoveryCacher struct {
	*HardwareCacher
	mac net.HardwareAddr
}

type DiscoveryTinkerbell struct {
	*HardwareTinkerbell
	mac net.HardwareAddr
}

type Interface interface {
	Name() string
}

type InterfaceCacher struct {
	*Port
}

type InterfaceTinkerbell struct {
	*DHCP
}

type Osie interface { // temp name

}

type OsieCacher struct {
	*ServicesVersion
}

type OsieTinkerbell struct {
	*Bootstrapper
}

type Hardware interface {
	HardwareAllowPXE() bool
	HardwareAllowWorkflow() bool
	HardwareArch() string
	HardwareBondingMode() BondingMode
	HardwareFacilityCode() string
	HardwareID() string
	HardwareIPs() []IP
	Interfaces() []Port
	HardwareManufacturer() string
	HardwarePlanSlug() string
	HardwarePlanVersionSlug() string
	HardwareState() HardwareState
	HardwareServicesVersion() Osie
	HardwareUEFI() bool
}

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

type HardwareTinkerbell struct {
	ID       string    `json:"id"`
	DHCP     DHCP      `json:"dhcp"`
	Netboot  Netboot   `json:"netboot"`
	Network  []Network `json:"network"`
	Metadata Metadata  `json:"metadata"`
}

// New instantiates a Discovery struct from the json argument
func NewDiscovery(j string) (*Discovery, error) {
	var res Discovery

	discoveryType := os.Getenv("DISCOVERY_TYPE")
	switch discoveryType {
	case discoveryTypeCacher:
		res = &DiscoveryCacher{}
	case discoveryTypeTinkerbell:
		res = &DiscoveryTinkerbell{}
	default:
		return nil, errors.New("invalid discovery type")
	}
	err := json.Unmarshal([]byte(j), &res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}
	return &res, err
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
	Storage ignition.Storage `json:"storage,omitempty"`
	SSHKeys []string         `json:"ssh_keys,omitempty"`
	// CustomData
	NetworkReady bool `json:"network_ready,omitempty"`
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

type DHCP struct {
	MAC         *MACAddr      `json:"mac"`
	IP          string        `json:"ip"`
	Hostname    string        `json:"hostname"`
	LeaseTime   time.Duration `json:"lease_time"`
	NameServers []string      `json:"name_servers"`
	TimeServers []string      `json:"time_servers"`
	Gateway     net.IP        `json:"gateway"`
	Arch        string        `json:"arch"`
	UEFI        bool          `json:"uefi"`
	IfaceName   string        `json:"iface_name"`
}

type Netboot struct {
	AllowPXE      bool `json:"allow_pxe"`
	AllowWorkflow bool `json:"allow_workflow"`
	IPXE          struct {
		URL      string `json:"url"`
		Contents string `json:"contents"`
	} `json:"ipxe"`
	Bootstrapper Bootstrapper `json:"bootstrapper"`
}

type Bootstrapper struct {
	Kernel string `json:"kernel"`
	Initrd string `json:"initrd"`
	OS     string `json:"os"`
}

type Network struct {
	DHCP    DHCP    `json:"dhcp,omitempty"`
	Netboot Netboot `json:"netboot,omitempty"`
}

type Metadata struct {
	State        HardwareState `json:"state"`
	BondingMode  BondingMode   `json:"bonding_mode"`
	Manufacturer Manufacturer  `json:"manufacturer"`
	Instance     *Instance     `json:"instance"`
	Custom       struct {
		PreinstalledOS OperatingSystem `json:"preinstalled_operating_system_version"`
		PrivateSubnets []string        `json:"private_subnets"`
	}
	Facility Facility `json:"facility"`
}

type Facility struct {
	PlanSlug        string `json:"plan_slug"`
	PlanVersionSlug string `json:"plan_version_slug"`
	FacilityCode    string `json:"facility_code"`
}
