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
		return h.HardwareUEFI(j.mac)
	}

	return false
}

func (j Job) Arch() string {
	if h := j.hardware; h != nil {
		return h.HardwareArch(j.mac)
	}

	return ""
}

func (j Job) BootDriveHint() string {
	if i := j.instance; i != nil {
		return i.BootDriveHint
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

// PasswordHash will return the password hash from the job instance if it exists
// PasswordHash first tries returning CryptedRootPassword if it exists and falls back to returning PasswordHash
func (j Job) PasswordHash() string {
	if j.instance == nil {
		return ""
	}
	// TODO: remove this EMism
	if j.instance.CryptedRootPassword != "" {
		return j.instance.CryptedRootPassword
	}

	return j.instance.PasswordHash
}

func (j Job) OperatingSystem() *packet.OperatingSystem {
	if i := j.instance; i != nil {
		if i.Rescue {
			return rescueOS
		}

		return j.hardware.OperatingSystem()
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

func (j Job) HardwareID() packet.HardwareID {
	if h := j.hardware; h != nil {
		return h.HardwareID()
	}

	return ""
}

func (j Job) FacilityCode() string {
	if h := j.hardware; h != nil {
		return h.HardwareFacilityCode()
	}

	return ""
}

func (j Job) PlanSlug() string {
	if h := j.hardware; h != nil {
		return h.HardwarePlanSlug()
	}

	return ""
}

func (j Job) PlanVersionSlug() string {
	if h := j.hardware; h != nil {
		return h.HardwarePlanVersionSlug()
	}

	return ""
}

func (j Job) Manufacturer() string {
	if h := j.hardware; h != nil {
		return h.HardwareManufacturer()
	}

	return ""
}

// PrimaryNIC returns the mac address of the NIC we expect to be dhcp/pxe'ing
func (j Job) PrimaryNIC() net.HardwareAddr {
	return j.mac
}

// HardwareState will return (enrolled burn_in preinstallable preinstalling failed_preinstall provisionable provisioning deprovisioning in_use)
func (j Job) HardwareState() string {
	if h := j.hardware; h != nil && h.HardwareID() != "" {
		return string(h.HardwareState())
	}

	return ""
}

// OSIEVersion returns any non-standard osie versions specified in either the instance proper or in userdata or attached to underlying hardware
func (j Job) OSIEVersion() string {
	if i := j.instance; i != nil {
		ov := i.ServicesVersion().OSIE
		if ov != "" {
			return ov
		}
	}
	h := j.hardware
	if h == nil {
		return ""
	}

	return h.HardwareOSIEVersion()
}

// CanWorkflow checks if workflow is allowed
func (j Job) CanWorkflow() bool {
	return j.hardware.HardwareAllowWorkflow(j.mac)
}

func (j Job) OSIEBaseURL() string {
	if h := j.hardware; h != nil {
		return j.hardware.OSIEBaseURL(j.mac)
	}

	return ""
}

func (j Job) KernelPath() string {
	if h := j.hardware; h != nil {
		return j.hardware.KernelPath(j.mac)
	}

	return ""
}

func (j Job) InitrdPath() string {
	if h := j.hardware; h != nil {
		return j.hardware.InitrdPath(j.mac)
	}

	return ""
}
