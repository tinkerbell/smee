package osie

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/google/go-cmp/cmp"
	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

func TestMain(m *testing.M) {
	logger, _ := log.Init("github.com/tinkerbell/boots")
	job.Init(logger)
	os.Exit(m.Run())
}

func genRandMAC(t *testing.T) string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	buf[0] = (buf[0] | 2) & 0xfe // Set local bit, ensure unicast address

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

var facility = func() string {
	fac := os.Getenv("FACILITY_CODE")
	if fac == "" {
		fac = "ewr1"
	}

	return fac
}()

func TestScript(t *testing.T) {
	for action, plan2Body := range action2Plan2Body {
		t.Run(action, func(t *testing.T) {
			for plan, body := range plan2Body {
				t.Run(plan, func(t *testing.T) {
					conf.OsieVendorServicesURL = "https://localhost"

					m := job.NewMock(t, plan, facility)
					m.SetManufacturer("supermicro")
					m.SetOSSlug("ubuntu_16_04_image")

					state := "provisioning"
					m.SetState(state)
					if action == "rescue" {
						m.SetRescue(true)
					}

					mac := genRandMAC(t)
					m.SetMAC(mac)

					s := ipxe.NewScript()
					s.Set("iface", "eth0")
					s.Or("shell")
					s.Set("tinkerbell", "http://127.0.0.1")
					s.Set("syslog_host", "127.0.0.1")
					s.Set("ipxe_cloud_config", "packet")

					Installer("", "", "", "", "", "", "", true).BootScript(action)(context.Background(), m.Job(), s)
					got := string(s.Bytes())

					arch := "aarch64"
					var parch string

					switch plan {
					case "baremetal_2a":
						parch = "aarch64"
					case "baremetal_2a2":
						parch = "2a2"
					case "baremetal_2a4":
						parch = "tx2"
					case "baremetal_2a5":
						parch = "qcom"
					case "baremetal_hua":
						parch = "hua"
					case "c2.large.arm", "c2.large.anbox":
						parch = "amp"
					case "c3.large.arm":
						parch = arch
					default:
						arch = "x86_64"
						parch = "x86_64"
					}

					preface := prefaces[action]
					preface = preface[:len(preface)-1] // drop extra \n at the end
					script := fmt.Sprintf(preface+body, action, state, arch, parch, mac)
					if script != got {
						t.Fatalf("%s bad iPXE script:\n%v", plan, diff.LineDiff(script, got))
					}
				})
			}
		})
	}
}

var prefaces = map[string]string{
	"discover": `#!ipxe

echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set syslog_host 127.0.0.1
set ipxe_cloud_config packet
set action %s
set state %s
set arch %s
set parch %s
set bootdevmac %s
set base-url http://install.ewr1.packet.net/misc/osie/current
`,
	"install": `#!ipxe

echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set syslog_host 127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set action %s
set state %s
set arch %s
set parch %s
set bootdevmac %s
`,
	"rescue": `#!ipxe

echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set syslog_host 127.0.0.1
set ipxe_cloud_config packet
set action %s
set state %s
set arch %s
set parch %s
set bootdevmac %s
set base-url http://install.` + facility + `.packet.net/misc/osie/current
`,
}

var action2Plan2Body = map[string]map[string]string{
	"discover": discoverBodies,
	"install":  installBodies,
	"rescue":   rescueBodies,
}

var discoverBodies = map[string]string{
	"c3.small.x86": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"c3.large.arm": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"baremetal_hua": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
}

var installBodies = map[string]string{
	"c3.small.x86": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=c3.small.x86 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"c3.large.arm": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 plan=c3.large.arm manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a2": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_2a2 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"baremetal_hua": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_hua manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"custom-osie": `
set base-url http://install.` + facility + `.packet.net/misc/osie/osie-v18.08.13.00
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_base_url=${base-url} packet_bootdev_mac=${bootdevmac} facility=ewr1 intel_iommu=on iommu=pt plan=custom-osie manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
}

var rescueBodies = map[string]string{
	"c3.small.x86": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"c3.large.arm": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"baremetal_hua": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
}

func TestOsieBaseURL(t *testing.T) {
	tests := map[string]struct {
		i    installer
		h    hardwareMock
		want string
	}{
		"OSIEBaseURL": {i: installer{}, h: hardwareMock{osieBaseURL: "http://my.custom.com/loc/for/osie"}, want: "http://my.custom.com/loc/for/osie"},
		"CustomOSIE":  {i: installer{osieURL: "http://my.url.com/files"}, h: hardwareMock{osieVersion: "v1.2.3"}, want: "http://my.url.com/files/v1.2.3"},
		"ipxeBaseURL": {i: installer{ipxeBaseURL: "http://example.com/files"}, h: hardwareMock{}, want: "http://example.com/files"},
		"ipxeBaseURL with trailing slash removed": {i: installer{ipxeBaseURL: "http://example.com/files/"}, h: hardwareMock{}, want: "http://example.com/files"},
		"osieURL":    {i: installer{osieURL: "http://www.example.com/files"}, h: hardwareMock{}, want: "http://www.example.com/files/current"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			j := job.NewMock(t, "", "", job.WithHardwareClient(tt.h))
			if diff := cmp.Diff(tt.want, tt.i.osieBaseURL(j.Job())); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

type hardwareMock struct {
	osieBaseURL string
	osieVersion string
}

func (h hardwareMock) HardwareAllowPXE(net.HardwareAddr) bool {
	return true
}

func (h hardwareMock) HardwareAllowWorkflow(mac net.HardwareAddr) bool {
	return true
}

func (h hardwareMock) HardwareArch(mac net.HardwareAddr) string {
	return ""
}

func (h hardwareMock) HardwareBondingMode() client.BondingMode {
	return client.BondingMode(0)
}

func (h hardwareMock) HardwareFacilityCode() string {
	return "1"
}

func (h hardwareMock) HardwareID() client.HardwareID {
	return client.HardwareID("")
}

func (h hardwareMock) HardwareIPs() []client.IP {
	return []client.IP{}
}

func (h hardwareMock) Interfaces() []client.Port {
	return []client.Port{}
}

func (h hardwareMock) HardwareManufacturer() string {
	return ""
}

func (h hardwareMock) HardwareProvisioner() string {
	return ""
}

func (h hardwareMock) HardwarePlanSlug() string {
	return ""
}

func (h hardwareMock) HardwarePlanVersionSlug() string {
	return ""
}

func (h hardwareMock) HardwareState() client.HardwareState {
	return ""
}

func (h hardwareMock) HardwareOSIEVersion() string {
	return h.osieVersion
}

func (h hardwareMock) HardwareUEFI(net.HardwareAddr) bool {
	return true
}

func (h hardwareMock) OSIEBaseURL(net.HardwareAddr) string {
	return h.osieBaseURL
}

func (h hardwareMock) KernelPath(net.HardwareAddr) string {
	return ""
}

func (h hardwareMock) InitrdPath(net.HardwareAddr) string {
	return ""
}

func (h hardwareMock) OperatingSystem() *client.OperatingSystem {
	return nil
}

func (h hardwareMock) GetTraceparent() string {
	return ""
}
