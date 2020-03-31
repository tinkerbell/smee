package coreos

import (
	"os"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/packethost/boots/ipxe"
	"github.com/packethost/boots/job"
)

var facility = func() string {
	fac := os.Getenv("FACILITY_CODE")
	if fac == "" {
		fac = "ewr1"
	}
	return fac
}()

func TestScript(t *testing.T) {
	for _, distro := range []string{"coreos", "flatcar"} {
		for typ, script := range type2Script {
			t.Run(typ+"-"+distro, func(t *testing.T) {
				m := job.NewMock(t, typ, facility)
				m.SetOSDistro(distro)

				s := ipxe.Script{}
				s.Echo("Packet.net Baremetal - iPXE boot")
				s.Set("iface", "eth0").Or("shell")
				s.Set("tinkerbell", "http://127.0.0.1")
				s.Set("ipxe_cloud_config", "packet")

				bootScript(m.Job(), &s)
				got := string(s.Bytes())
				script = strings.Replace(script, "coreos", distro, -1)
				if script != got {
					t.Fatalf("%s bad iPXE script:\n%v", typ, diff.LineDiff(script, got))
				}
			})
		}
	}
}

var type2Script = map[string]string{
	"baremetal_0": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos_production_pxe.vmlinuz console=ttyS1,115200n8 console=tty0 vga=773 initrd=coreos_production_pxe_image.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos_production_pxe_image.cpio.gz
boot
`,
	"baremetal_1": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos_production_pxe.vmlinuz console=ttyS1,115200n8 console=tty0 vga=773 initrd=coreos_production_pxe_image.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos_production_pxe_image.cpio.gz
boot
`,
	"baremetal_2": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos_production_pxe.vmlinuz console=ttyS1,115200n8 console=tty0 vga=773 initrd=coreos_production_pxe_image.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos_production_pxe_image.cpio.gz
boot
`,
	"baremetal_3": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos_production_pxe.vmlinuz console=ttyS1,115200n8 console=tty0 vga=773 initrd=coreos_production_pxe_image.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos_production_pxe_image.cpio.gz
boot
`,
	"baremetal_2a": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos-arm.vmlinuz console=ttyAMA0,115200 initrd=coreos-arm.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos-arm.cpio.gz
boot
`,
	"baremetal_2a2": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos-arm.vmlinuz console=ttyAMA0,115200 initrd=coreos-arm.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos-arm.cpio.gz
boot
`,
	"baremetal_hua": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/coreos-arm.vmlinuz console=ttyAMA0,115200 initrd=coreos-arm.cpio.gz bonding.max_bonds=0 coreos.autologin coreos.first_boot=1 coreos.config.url=${tinkerbell}/coreos/ignition.json systemd.setenv=oem_url=${tinkerbell}/coreos/oem.tgz systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/coreos-arm.cpio.gz
boot
`,
}
