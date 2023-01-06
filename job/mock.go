package job

import (
	"context"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/standalone"
	"go.uber.org/zap/zaptest"
)

type Mock Job

// NewMock returns a mock Job with only minimal fields set, it is useful only for tests.
func NewMock(t zaptest.TestingT, slug, facility string) Mock {
	slugs := strings.Split(slug, ":")
	slug = slugs[0]
	var planVersion string
	if len(slugs) > 1 {
		planVersion = slugs[1]
	}

	arch := "x86_64"
	if strings.Contains(slug, ".arm") || strings.Contains(slug, "baremetal_2a") || strings.Contains(slug, "baremetal_hua") {
		arch = "aarch64"
	}

	uefi := false
	if arch == "aarch64" || slug == "c2.medium.x86" {
		uefi = true
	}

	servicesVersion := client.ServicesVersion{}
	if strings.Contains(slug, "custom-osie") {
		servicesVersion.OSIE = "osie-v18.08.13.00"
	}

	mockLog := log.Test(t, "job.Mock")

	return Mock{
		Logger: mockLog.With("mock", true, "slug", slug, "arch", arch, "uefi", uefi),
		hardware: &standalone.HardwareStandalone{
			ID: uuid.New().String(),
			Metadata: client.Metadata{
				Facility: client.Facility{
					PlanSlug:        slug,
					PlanVersionSlug: planVersion,
					FacilityCode:    facility,
				},
				Instance: &client.Instance{
					OS: &client.OperatingSystem{},
				},
				State: "provisionable",
			},
			Network: client.Network{
				Interfaces: []client.NetworkInterface{
					{
						DHCP: client.DHCP{
							UEFI: uefi,
							Arch: arch,
						},
					},
				},
			},
		},
		instance: &client.Instance{
			State:           "provisioning",
			ServicesVersion: servicesVersion,
		},
	}
}

func NewMockFromDiscovery(d client.Discoverer, mac net.HardwareAddr) Mock {
	mockLog, _ := log.Init("job.Mock")
	j := Job{Logger: mockLog, mac: mac}
	_, _ = j.setup(context.Background(), d)

	return Mock(j)
}

func (m Mock) Job() Job {
	return Job(m)
}

func (m *Mock) DropInstance() {
	m.instance = nil
}

func (m *Mock) SetIP(ip net.IP) {
	m.ip = ip
}

func (m *Mock) SetIPXEScriptURL(url string) {
	m.instance.IPXEScriptURL = url
}

func (m *Mock) SetUserData(userdata string) {
	m.instance.UserData = userdata
}

func (m *Mock) SetMAC(mac string) {
	_m, err := net.ParseMAC(mac)
	if err != nil {
		panic(err)
	}
	m.mac = _m
}

func (m *Mock) SetManufacturer(slug string) {
	hp := m.hardware
	h, ok := hp.(*standalone.HardwareStandalone)
	if ok {
		h.Metadata.Manufacturer = client.Manufacturer{Slug: slug}
	}
}

func (m *Mock) SetOSDistro(distro string) {
	m.hardware.OperatingSystem().Distro = distro
}

func (m *Mock) SetOSSlug(slug string) {
	m.hardware.OperatingSystem().Slug = slug
	m.hardware.OperatingSystem().OsSlug = slug
}

func (m *Mock) SetOSVersion(version string) {
	m.hardware.OperatingSystem().Version = version
}

func (m *Mock) SetOSImageTag(tag string) {
	m.hardware.OperatingSystem().ImageTag = tag
}

func (m *Mock) SetOSInstaller(installer string) {
	m.hardware.OperatingSystem().Installer = installer
}

func (m *Mock) SetOSInstallerData(installerData *client.InstallerData) {
	m.hardware.OperatingSystem().InstallerData = installerData
}

func (m *Mock) SetPassword(string) {
	m.instance.CryptedRootPassword = "insecure"
	m.instance.PasswordHash = "insecure"
}

func (m *Mock) SetCustomData(data interface{}) {
	m.instance.CustomData = data
}

func (m *Mock) SetState(state string) {
	hp := m.hardware
	h, ok := hp.(*standalone.HardwareStandalone)
	if ok {
		h.Metadata.State = client.HardwareState(state)
	}
}

func (m *Mock) SetBootDriveHint(drive string) {
	m.instance.BootDriveHint = drive
}

func (m *Mock) SetRescue(b bool) {
	i := m.instance
	i.Rescue = b
}
