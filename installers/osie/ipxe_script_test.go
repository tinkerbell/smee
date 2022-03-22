package osie

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

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

					state := ""
					if action == "install" {
						state = "provisioning"
					} else {
						state = "rescuing"
					}
					m.SetState(state)

					mac := genRandMAC(t)
					m.SetMAC(mac)

					s := ipxe.Script{}
					s.Echo("Tinkerbell Boots iPXE")
					s.Set("iface", "eth0").Or("shell")
					s.Set("tinkerbell", "http://127.0.0.1")
					s.Set("syslog_host", "127.0.0.1")
					s.Set("ipxe_cloud_config", "packet")
					o := Installer{}
					ctx := context.Background()
					var bs ipxe.Script
					switch action {
					case "rescue":
						bs = o.Rescue()(ctx, m.Job(), s)
					case "install":
						bs = o.Install()(ctx, m.Job(), s)
					case "discover":
						bs = o.Discover()(ctx, m.Job(), s)
					}
					got := string(bs.Bytes())

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
					case "c2.large.arm", "c2.large.anbox", "c3.large.arm":
						parch = "amp"
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
	"discover": `echo Tinkerbell Boots iPXE
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
	"install": `echo Tinkerbell Boots iPXE
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
	"rescue": `echo Tinkerbell Boots iPXE
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
	"baremetal_0": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_1": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_3": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"baremetal_2a4": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a5": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_s": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_hua": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"c2.large.arm": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
}

var installBodies = map[string]string{
	"baremetal_0": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_0 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_1": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_1 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_2 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_3": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_3 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_2a manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
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
	"baremetal_2a4": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_2a4 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a5": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_2a5 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_s": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_s manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_hua": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=baremetal_hua manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"c2.large.arm": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 plan=c2.large.arm manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"c2.medium.x86": `
set base-url http://install.` + facility + `.packet.net/misc/osie/current
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt plan=c2.medium.x86 manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"custom-osie": `
set base-url http://install.` + facility + `.packet.net/misc/osie/osie-v18.08.13.00
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_base_url=http://install.ewr1.packet.net/misc/osie/osie-v18.08.13.00 packet_bootdev_mac=${bootdevmac} facility=ewr1 intel_iommu=on iommu=pt plan=custom-osie manufacturer=supermicro slug=ubuntu_16_04 initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
}

var rescueBodies = map[string]string{
	"baremetal_0": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_1": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_3": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a2": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"baremetal_2a4": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_2a5": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_s": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
	"baremetal_hua": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` intel_iommu=on iommu=pt initrd=initramfs-${parch} console=ttyS0,115200
initrd ${base-url}/initramfs-${parch}
sleep 15
boot
`,
	"c2.large.arm": `
kernel ${base-url}/vmlinuz-${parch} ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${parch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} parch=${parch} packet_action=${action} packet_state=${state} osie_vendors_url=https://localhost packet_bootdev_mac=${bootdevmac} facility=` + facility + ` iommu.passthrough=1 initrd=initramfs-${parch} console=ttyAMA0,115200
initrd ${base-url}/initramfs-${parch}
boot
`,
}
