package job

import (
	"net"

	"github.com/tinkerbell/boots/packet"
)

var rescueOS = &packet.OperatingSystem{
	Slug:    "alpine_3",
	Distro:  "alpine",
	Version: "3",
}

func (j Job) IsARM() bool {
	return j.Arch() == "aarch64"
}

func (j Job) IsUEFI() bool {
	if h := j.hardware; h != nil {
		return h.UEFI
	}
	return false
}

func (j Job) Arch() string {
	if h := j.hardware; h != nil {
		return h.Arch
	}
	return ""
}

func (j Job) PArch() string {
	var parch string

	switch j.PlanSlug() {
	case "baremetal_2a2", "c1.large.arm.xda":
		parch = "2a2"
	case "baremetal_2a4":
		parch = "tx2"
	case "baremetal_2a5":
		parch = "qcom"
	case "baremetal_hua":
		parch = "hua"
	case "c2.large.arm", "c2.large.anbox":
		parch = "amp"
	}

	if parch != "" {
		return parch
	}

	return j.Arch()
}

func (j Job) InstanceID() string {
	if i := j.instance; i != nil {
		return i.ID
	}
	return ""
}

// UserData returns instance.UserData
func (j Job) UserData() string {
	if i := j.instance; i != nil {
		return i.UserData
	}
	return ""
}

// IPXEScriptURL returns the value of instance.IPXEScriptURL
func (j Job) IPXEScriptURL() string {
	if i := j.instance; i != nil {
		return i.IPXEScriptURL
	}
	return ""
}

func (j Job) InstanceIPs() []packet.IP {
	if i := j.instance; i != nil {
		return i.IPs
	}
	return nil
}

func (j Job) PrivateSubnets() []string {
	if h := j.hardware; h != nil {
		return h.PrivateSubnets
	}
	return nil
}

func (j Job) CryptedPassword() string {
	if j.instance != nil {
		return j.instance.CryptedRootPassword
	}
	return ""
}

func (j Job) OperatingSystem() *packet.OperatingSystem {
	if i := j.instance; i != nil {
		if i.Rescue {
			return rescueOS
		}
		return &i.OS
	}
	return nil
}

func (j Job) ID() string {
	return j.mac.String()
}

func (j Job) Interfaces() []packet.Port {
	if h := j.hardware; h != nil {
		return h.Interfaces()
	}
	return nil
}

func (j Job) InterfaceName(i int) string {
	if ifaces := j.Interfaces(); len(ifaces) > i {
		return ifaces[i].Name
	}
	return ""
}

func (j Job) InterfaceMAC(i int) net.HardwareAddr {
	if ifaces := j.Interfaces(); len(ifaces) > i {
		return ifaces[i].MAC()
	}
	return nil
}

func (j Job) HardwareID() string {
	if h := j.hardware; h != nil {
		return h.ID
	}
	return ""
}

func (j Job) FacilityCode() string {
	if h := j.hardware; h != nil {
		return h.FacilityCode
	}
	return ""
}

func (j Job) PlanSlug() string {
	if h := j.hardware; h != nil {
		return h.PlanSlug
	}
	return ""
}

func (j Job) PlanVersionSlug() string {
	if h := j.hardware; h != nil {
		return h.PlanVersionSlug
	}
	return ""
}

func (j Job) Manufacturer() string {
	if h := j.hardware; h != nil {
		return h.Manufacturer.Slug
	}
	return ""
}

// PrimaryNIC returns the mac address of the NIC we expect to be dhcp/pxe'ing
func (j Job) PrimaryNIC() net.HardwareAddr {
	return j.mac
}

// golangci-lint: unused
//func (j Job) isPrimaryNIC(mac net.HardwareAddr) bool {
//	return bytes.Equal(mac, j.PrimaryNIC())
//}

// HardwareState will return (enrolled burn_in preinstallable preinstalling failed_preinstall provisionable provisioning deprovisioning in_use)
func (j Job) HardwareState() string {
	if h := j.hardware; h != nil && h.ID != "" {
		return string(h.State)
	}
	return ""
}

func (j Job) ServicesVersion() packet.ServicesVersion {
	if h := j.hardware; h != nil && h.ID != "" {
		return h.ServicesVersion
	}
	return packet.ServicesVersion{}
}
