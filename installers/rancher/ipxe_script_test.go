package rancher

import (
	"context"
	"os"
	"testing"

	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

var facility = func() string {
	fac := os.Getenv("FACILITY_CODE")
	if fac == "" {
		fac = "ewr1"
	}

	return fac
}()

func TestScript(t *testing.T) {
	for typ, script := range type2Script {
		t.Run(typ, func(t *testing.T) {
			m := job.NewMock(t, typ, facility)

			s := ipxe.Script{}
			s.Echo("Tinkerbell Boots iPXE")
			s.Set("iface", "eth0").Or("shell")
			s.Set("tinkerbell", "http://127.0.0.1")
			s.Set("ipxe_cloud_config", "packet")
			r := Installer{}
			bs := r.BootScript()(context.Background(), m.Job(), s)
			got := string(bs.Bytes())
			if script != got {
				t.Fatalf("%s bad iPxe script\nwant:\n%s\ngot:\n%s", typ, script, got)
			}
		})
	}
}

var type2Script = map[string]string{
	"baremetal_0": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] rancher.network.interfaces.eth0.dhcp=true rancher.network.interfaces.eth2.dhcp=true tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_1": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] rancher.network.interfaces.eth0.dhcp=true rancher.network.interfaces.eth2.dhcp=true tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_2": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_3": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_2a": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_2a2": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
	"baremetal_hua": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://releases.rancher.com/os/packet
kernel ${base-url}/vmlinuz console=ttyS1,115200n8 rancher.cloud_init.datasources=[url:${base-url}/packet.sh] tinkerbell=${tinkerbell}
initrd ${base-url}/initrd
boot
`,
}
