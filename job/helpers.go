package job

import (
	"net"

	"github.com/tinkerbell/boots/client"
)

var rescueOS = &client.OperatingSystem{
	Slug:    "alpine_3",
	Distro:  "alpine",
	Version: "3",
}

func (j *Job) IsUEFI() bool {
	if h := j.hardware; h != nil {
		return h.HardwareUEFI(j.mac)
	}

	return false
}

func (j *Job) Arch() string {
	if h := j.hardware; h != nil {
		return h.HardwareArch(j.mac)
	}

	return ""
}

func (j *Job) InstanceID() string {
	if i := j.instance; i != nil {
		return i.ID
	}

	return ""
}

func (j *Job) OperatingSystem() *client.OperatingSystem {
	if i := j.instance; i != nil {
		if i.Rescue {
			return rescueOS
		}

		return j.hardware.OperatingSystem()
	}

	return nil
}

func (j *Job) FacilityCode() string {
	if h := j.hardware; h != nil {
		return h.HardwareFacilityCode()
	}

	return ""
}

// PrimaryNIC returns the mac address of the NIC we expect to be dhcp/pxe'ing.
func (j *Job) PrimaryNIC() net.HardwareAddr {
	return j.mac
}

// HardwareState will return (enrolled burn_in preinstallable preinstalling failed_preinstall provisionable provisioning deprovisioning in_use).
func (j *Job) HardwareState() string {
	if h := j.hardware; h != nil && h.HardwareID() != "" {
		return string(h.HardwareState())
	}

	return ""
}

// OSIEVersion returns any non-standard osie versions specified in either the instance proper or in userdata or attached to underlying hardware.
func (j *Job) OSIEVersion() string {
	if i := j.instance; i != nil {
		ov := i.GetServicesVersion().OSIE
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

func (j *Job) OSIEBaseURL() string {
	if h := j.hardware; h != nil {
		return j.hardware.OSIEBaseURL(j.mac)
	}

	return ""
}
