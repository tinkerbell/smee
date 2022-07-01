package client

import (
	"bufio"
	"encoding/json"
	"net"
	"regexp"
	"strings"
)

var servicesVersionUserdataRegex = regexp.MustCompile(`^\s*#\s*services\s*=\s*({.*})\s*$`)

// Instance models the instance data as returned by the API.
type Instance struct {
	ID       string        `json:"id"`
	State    InstanceState `json:"state"`
	Hostname string        `json:"hostname"`
	AllowPXE bool          `json:"allow_pxe"`
	Rescue   bool          `json:"rescue"`

	OS              *OperatingSystem `json:"operating_system"`
	OSV             *OperatingSystem `json:"operating_system_version"`
	AlwaysPXE       bool             `json:"always_pxe,omitempty"`
	IPXEScriptURL   string           `json:"ipxe_script_url,omitempty"`
	IPs             []IP             `json:"ip_addresses"`
	UserData        string           `json:"userdata,omitempty"`
	servicesVersion ServicesVersion

	// Same as PasswordHash
	// Duplicated here, because CryptedRootPassword is in cacher/legacy mode
	// which is soon to go away as Tinkerbell/PasswordHash is the future
	CryptedRootPassword string `json:"crypted_root_password,omitempty"`
	// Only returned in the first 24 hours of a provision
	PasswordHash string `json:"password_hash,omitempty"`

	Tags []string `json:"tags,omitempty"`
	// Project
	SSHKeys []string `json:"ssh_keys,omitempty"`
	// CustomData
	NetworkReady bool `json:"network_ready,omitempty"`
	// BootDriveHint defines what the VMware installer should pass as the argument to "--firstdisk=".
	BootDriveHint string `json:"boot_drive_hint,omitempty"`
}

// Device Full device result from /devices endpoint.
type Device struct {
	ID string `json:"id"`
}

// FindIP returns IP for an instance, nil otherwise.
func (i *Instance) FindIP(pred func(IP) bool) *IP {
	for _, ip := range i.IPs {
		if pred(ip) {
			return &ip
		}
	}

	return nil
}

func (i *Instance) ServicesVersion() ServicesVersion {
	if i.servicesVersion.OSIE != "" {
		return i.servicesVersion
	}

	if i.UserData == "" {
		return ServicesVersion{}
	}

	scanner := bufio.NewScanner(strings.NewReader(i.UserData))
	for scanner.Scan() {
		matches := servicesVersionUserdataRegex.FindStringSubmatch(scanner.Text())
		if len(matches) == 0 {
			continue
		}

		var sv ServicesVersion
		err := json.Unmarshal([]byte(matches[1]), &sv)
		if err != nil {
			return ServicesVersion{}
		}

		return sv
	}

	return ServicesVersion{}
}

func ManagementPublicIPv4IP(ip IP) bool {
	return ip.Public && ip.Management && ip.Family == 4
}

func ManagementPrivateIPv4IP(ip IP) bool {
	return !ip.Public && ip.Management && ip.Family == 4
}

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
	OSIE string `json:"osie"`
}

// IP represents IP address for a hardware.
type IP struct {
	Address    net.IP `json:"address"`
	Netmask    net.IP `json:"netmask"`
	Gateway    net.IP `json:"gateway"`
	Family     int    `json:"address_family"`
	Public     bool   `json:"public"`
	Management bool   `json:"management"`
}

// OperatingSystem holds details for the operating system.
type OperatingSystem struct {
	Slug          string         `json:"slug"`
	Distro        string         `json:"distro"`
	Version       string         `json:"version"`
	ImageTag      string         `json:"image_tag"`
	OsSlug        string         `json:"os_slug"`
	Installer     string         `json:"installer,omitempty"`
	InstallerData *InstallerData `json:"installer_data,omitempty"`
}

// InstallerData holds a number of fields that may be used by an installer.
type InstallerData struct {
	Chain  string `json:"chain,omitempty"`
	Script string `json:"script,omitempty"`
}

// Port represents a network port.
type Port struct {
	ID   string   `json:"id"`
	Type PortType `json:"type"`
	Name string   `json:"name"`
	Data struct {
		MAC  *MACAddr `json:"mac"`
		Bond string   `json:"bond"`
	} `json:"data"`
}

// MAC returns the physical hardware address, nil otherwise.
func (p *Port) MAC() net.HardwareAddr {
	if p.Data.MAC != nil && *p.Data.MAC != ZeroMAC {
		return p.Data.MAC.HardwareAddr()
	}

	return nil
}

// PortType is type for a network port.
type PortType string

// Manufacturer holds data for hardware manufacturer.
type Manufacturer struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type NetworkInterface struct {
	DHCP    DHCP    `json:"dhcp,omitempty"`
	Netboot Netboot `json:"netboot,omitempty"`
}

// DHCP holds details for DHCP connection.
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

// Netboot holds details for a hardware to boot over network.
type Netboot struct {
	AllowPXE      bool `json:"allow_pxe"`      // to be removed?
	AllowWorkflow bool `json:"allow_workflow"` // to be removed?
	IPXE          struct {
		URL      string `json:"url"`
		Contents string `json:"contents"`
	} `json:"ipxe"`
	OSIE OSIE `json:"osie"`
}

// Bootstrapper is the bootstrapper to be used during netboot.
type OSIE struct {
	BaseURL string `json:"base_url"`
	Kernel  string `json:"kernel"`
	Initrd  string `json:"initrd"`
}

// Network holds hardware network details.
type Network struct {
	Interfaces []NetworkInterface `json:"interfaces,omitempty"`
}

// InterfacesByMac returns the NetworkInterface that contains the matching mac address
// returns an empty NetworkInterface if not found.
func (n Network) InterfaceByMac(mac net.HardwareAddr) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.MAC.String() == mac.String() {
			return i
		}
	}

	return NetworkInterface{}
}

// InterfacesByIp returns the NetworkInterface that contains the matching ip address
// returns an empty NetworkInterface if not found.
func (n Network) InterfaceByIP(ip net.IP) NetworkInterface {
	for _, i := range n.Interfaces {
		if i.DHCP.IP.Address.String() == ip.String() {
			return i
		}
	}

	return NetworkInterface{}
}

// Metadata holds the hardware metadata.
type Metadata struct {
	State        HardwareState `json:"state"`
	BondingMode  BondingMode   `json:"bonding_mode"`
	Manufacturer Manufacturer  `json:"manufacturer"`
	Instance     *Instance     `json:"instance"`
	Custom       struct {
		PreinstalledOS OperatingSystem `json:"preinstalled_operating_system_version"`
		PrivateSubnets []string        `json:"private_subnets"`
	} `json:"custom"`
	Facility          Facility `json:"facility"`
	ProvisionerEngine string   `json:"provisioner_engine"`
}

// Facility represents the facilty in use.
type Facility struct {
	PlanSlug        string `json:"plan_slug"`
	PlanVersionSlug string `json:"plan_version_slug"`
	FacilityCode    string `json:"facility_code"`
}
