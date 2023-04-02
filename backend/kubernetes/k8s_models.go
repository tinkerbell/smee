package kubernetes

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
)

func tinkOsToDiscovererOS(in *v1alpha1.MetadataInstanceOperatingSystem) *backend.OperatingSystem {
	if in == nil {
		return nil
	}

	return &backend.OperatingSystem{
		Slug:     in.Slug,
		Distro:   in.Distro,
		Version:  in.Version,
		ImageTag: in.ImageTag,
		OsSlug:   in.OsSlug,
	}
}

func tinkIPToDiscovererIP(in *v1alpha1.MetadataInstanceIP) *backend.IP {
	if in == nil {
		return nil
	}

	return &backend.IP{
		Address:    net.ParseIP(in.Address),
		Netmask:    net.ParseIP(in.Netmask),
		Gateway:    net.ParseIP(in.Gateway),
		Family:     int(in.Family),
		Public:     in.Public,
		Management: in.Management,
	}
}

func (d *K8sDiscoverer) Instance() *backend.Instance {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Instance != nil {
		return &backend.Instance{
			ID:            d.hw.Spec.Metadata.Instance.ID,
			State:         backend.InstanceState(d.hw.Spec.Metadata.Instance.State),
			Hostname:      d.hw.Spec.Metadata.Instance.Hostname,
			AllowPXE:      d.hw.Spec.Metadata.Instance.AllowPxe,
			Rescue:        d.hw.Spec.Metadata.Instance.Rescue,
			OS:            tinkOsToDiscovererOS(d.hw.Spec.Metadata.Instance.OperatingSystem),
			AlwaysPXE:     d.hw.Spec.Metadata.Instance.AlwaysPxe,
			IPXEScriptURL: d.hw.Spec.Metadata.Instance.IpxeScriptURL,
			IPs: func(in []*v1alpha1.MetadataInstanceIP) []backend.IP {
				resp := []backend.IP{}
				for _, ip := range in {
					resp = append(resp, *tinkIPToDiscovererIP(ip))
				}

				return resp
			}(d.hw.Spec.Metadata.Instance.Ips),
			UserData:            d.hw.Spec.Metadata.Instance.Userdata,
			CryptedRootPassword: d.hw.Spec.Metadata.Instance.CryptedRootPassword,
			Tags:                d.hw.Spec.Metadata.Instance.Tags,
			SSHKeys:             d.hw.Spec.Metadata.Instance.SSHKeys,
			NetworkReady:        d.hw.Spec.Metadata.Instance.NetworkReady,
		}
	}

	return nil
}

func (d *K8sDiscoverer) MAC() net.HardwareAddr {
	if len(d.hw.Spec.Interfaces) > 0 && d.hw.Spec.Interfaces[0].DHCP != nil {
		mac, err := net.ParseMAC(d.hw.Spec.Interfaces[0].DHCP.MAC)
		if err != nil {
			return nil
		}

		return mac
	}

	return nil
}

func (d *K8sDiscoverer) Mode() string {
	return "hardware"
}

func (d *K8sDiscoverer) GetIP(addr net.HardwareAddr) backend.IP {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.MAC != "" && iface.DHCP.IP != nil {
			if addr.String() == iface.DHCP.MAC {
				return backend.IP{
					Address: net.ParseIP(iface.DHCP.IP.Address),
					Netmask: net.ParseIP(iface.DHCP.IP.Netmask),
					Gateway: net.ParseIP(iface.DHCP.IP.Gateway),
					Family:  int(iface.DHCP.IP.Family),
					// TODO not 100% accurate
					Public: !net.ParseIP(iface.DHCP.IP.Address).IsPrivate(),
					// TODO: When should we set this to true?
					Management: false,
				}
			}
		}
	}

	return backend.IP{}
}

func (d *K8sDiscoverer) GetMAC(ip net.IP) net.HardwareAddr {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.MAC != "" && iface.DHCP.IP != nil {
			if ip.String() == iface.DHCP.IP.Address {
				mac, err := net.ParseMAC(iface.DHCP.MAC)
				if err != nil {
					return nil
				}

				return mac
			}
		}
	}

	return nil
}

func (d *K8sDiscoverer) DNSServers(net.HardwareAddr) []net.IP {
	resp := []net.IP{}
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.MAC != "" {
			for _, ns := range iface.DHCP.NameServers {
				resp = append(resp, net.ParseIP(ns))
			}
		}
	}

	return resp
}

func (d *K8sDiscoverer) LeaseTime(net.HardwareAddr) time.Duration {
	if len(d.hw.Spec.Interfaces) > 0 && d.hw.Spec.Interfaces[0].DHCP != nil {
		return time.Duration(d.hw.Spec.Interfaces[0].DHCP.LeaseTime) * time.Second
	}
	// Default to 24 hours?

	return time.Hour * 24
}

func (d *K8sDiscoverer) Hostname() (string, error) {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.Hostname != "" {
			return iface.DHCP.Hostname, nil
		}
	}

	return "", nil
}

func (d *K8sDiscoverer) Hardware() backend.Hardware { return d }

func (d *K8sDiscoverer) SetMAC(net.HardwareAddr) {}

func NewK8sDiscoverer(hw *v1alpha1.Hardware) backend.Discoverer {
	return &K8sDiscoverer{hw: hw}
}

type K8sDiscoverer struct {
	hw *v1alpha1.Hardware
}

var (
	_ backend.Discoverer = &K8sDiscoverer{}
	_ backend.Hardware   = &K8sDiscoverer{}
)

func (d *K8sDiscoverer) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.DHCP != nil && mac.String() == iface.DHCP.MAC {
			return *iface.Netboot.AllowWorkflow
		}
	}

	return false
}

func (d *K8sDiscoverer) HardwareAllowPXE(mac net.HardwareAddr) bool {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.DHCP != nil && mac.String() == iface.DHCP.MAC {
			return *iface.Netboot.AllowPXE
		}
	}

	return false
}

func (d *K8sDiscoverer) HardwareArch(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil {
			return iface.DHCP.Arch
		}
	}

	return ""
}

func (d *K8sDiscoverer) HardwareBondingMode() backend.BondingMode {
	if d.hw.Spec.Metadata != nil {
		return backend.BondingMode(d.hw.Spec.Metadata.BondingMode)
	}

	return backend.BondingMode(0)
}

func (d *K8sDiscoverer) HardwareFacilityCode() string {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Facility != nil {
		return d.hw.Spec.Metadata.Facility.FacilityCode
	}

	return ""
}

func (d *K8sDiscoverer) HardwareID() backend.HardwareID {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Instance != nil {
		return backend.HardwareID(d.hw.Spec.Metadata.Instance.ID)
	}

	return backend.HardwareID("")
}

func (d *K8sDiscoverer) HardwareIPs() []backend.IP {
	resp := []backend.IP{}
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Instance != nil {
		for _, ip := range d.hw.Spec.Metadata.Instance.Ips {
			resp = append(resp, *tinkIPToDiscovererIP(ip))
		}
	}

	return resp
}

func (d *K8sDiscoverer) Interfaces() []backend.Port { return nil }

func (d *K8sDiscoverer) HardwareManufacturer() string {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Manufacturer != nil {
		return d.hw.Spec.Metadata.Manufacturer.ID
	}

	return ""
}

func (d *K8sDiscoverer) HardwareProvisioner() string {
	return ""
}

func (d *K8sDiscoverer) HardwarePlanSlug() string {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Facility != nil {
		return d.hw.Spec.Metadata.Facility.PlanSlug
	}

	return ""
}

func (d *K8sDiscoverer) HardwarePlanVersionSlug() string {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Facility != nil {
		return d.hw.Spec.Metadata.Facility.PlanVersionSlug
	}

	return ""
}

func (d *K8sDiscoverer) HardwareState() backend.HardwareState {
	if d.hw.Spec.Metadata != nil {
		return backend.HardwareState(d.hw.Spec.Metadata.State)
	}

	return ""
}

func (d *K8sDiscoverer) HardwareOSIEVersion() string {
	return ""
}

func (d *K8sDiscoverer) HardwareUEFI(net.HardwareAddr) bool {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil {
			return iface.DHCP.UEFI
		}
	}

	return false
}

func (d *K8sDiscoverer) OSIEBaseURL(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.Netboot.OSIE != nil {
			return iface.Netboot.OSIE.BaseURL
		}
	}

	return ""
}

func (d *K8sDiscoverer) KernelPath(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.Netboot.OSIE != nil {
			return iface.Netboot.OSIE.Kernel
		}
	}

	return ""
}

func (d *K8sDiscoverer) InitrdPath(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.Netboot.OSIE != nil {
			return iface.Netboot.OSIE.Initrd
		}
	}

	return ""
}

func (d *K8sDiscoverer) IPXEURL(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.Netboot.IPXE != nil {
			return iface.Netboot.IPXE.URL
		}
	}

	return ""
}

func (d *K8sDiscoverer) IPXEScript(net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.Netboot != nil && iface.Netboot.IPXE != nil {
			return iface.Netboot.IPXE.Contents
		}
	}

	return ""
}

func (d *K8sDiscoverer) OperatingSystem() *backend.OperatingSystem {
	if d.hw.Spec.Metadata != nil && d.hw.Spec.Metadata.Instance != nil && d.hw.Spec.Metadata.Instance.OperatingSystem != nil {
		return &backend.OperatingSystem{
			Slug:     d.hw.Spec.Metadata.Instance.OperatingSystem.Slug,
			Distro:   d.hw.Spec.Metadata.Instance.OperatingSystem.Distro,
			Version:  d.hw.Spec.Metadata.Instance.OperatingSystem.Version,
			ImageTag: d.hw.Spec.Metadata.Instance.OperatingSystem.ImageTag,
			OsSlug:   d.hw.Spec.Metadata.Instance.OperatingSystem.OsSlug,
		}
	}

	return nil
}

// GetTraceparent always returns an empty string.
func (d *K8sDiscoverer) GetTraceparent() string {
	return ""
}

// GetVLANID gets the VLAN ID for the given MAC address.
func (d *K8sDiscoverer) GetVLANID(mac net.HardwareAddr) string {
	for _, iface := range d.hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.MAC == mac.String() {
			return iface.DHCP.VLANID
		}
	}

	return ""
}
