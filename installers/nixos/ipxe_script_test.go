package nixos

import (
	"os"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
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

			split := strings.Split(typ, "/")
			v, typ := split[0], split[1]
			tag := ""
			if strings.Contains(typ, ":") {
				split = strings.Split(typ, ":")
				typ, tag = split[0], split[1]
			}

			m := job.NewMock(t, typ, facility)
			m.SetOSSlug("nixos_" + v)
			if tag != "" {
				m.SetOSImageTag(tag)
			}

			s := ipxe.Script{}
			s.Echo("Packet.net Baremetal - iPXE boot")
			s.Set("iface", "eth0").Or("shell")
			s.Set("tinkerbell", "http://127.0.0.1")
			s.Set("ipxe_cloud_config", "packet")

			bootScript(oshwToInitPath, m.Job(), &s)
			got := string(s.Bytes())
			if script != got {
				t.Fatalf("%s bad iPXE script:\n%v", typ, diff.LineDiff(script, got))
			}
		})
	}
}

var type2Script = map[string]string{
	"17_03/t1.small.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_17_03/t1.small.x86
kernel ${base-url}/kernel init=/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"17_03/c1.small.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_17_03/c1.small.x86
kernel ${base-url}/kernel init=/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"17_03/m1.xlarge.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_17_03/m1.xlarge.x86
kernel ${base-url}/kernel init=/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"17_03/c1.xlarge.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_17_03/c1.xlarge.x86
kernel ${base-url}/kernel init=/nix/store/a8nhjab9brxw80lnvrpxj37wkgmxa0bl-nixos-system-ipxe-17.03.945.5acb454e2a/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/t1.small.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/t1.small.x86
kernel ${base-url}/kernel init=/nix/store/9zpihimwsjysscvidjs1dfa0zwfnxim0-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/c1.small.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/c1.small.x86
kernel ${base-url}/kernel init=/nix/store/hq6hni37qjql1206j7hkhqf1x017w8qz-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/c2.medium.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/c2.medium.x86
kernel ${base-url}/kernel init=/nix/store/46mmhc2jv2wkkda90dqks1p2054irszy-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/m1.xlarge.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/m1.xlarge.x86
kernel ${base-url}/kernel init=/nix/store/59v36skcl0ymsq61phx5yxifn89ddi9n-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/m2.xlarge.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/m2.xlarge.x86
kernel ${base-url}/kernel init=/nix/store/zizskvd3hb9arcn7lswqy1j81p538q1w-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/c1.xlarge.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/c1.xlarge.x86
kernel ${base-url}/kernel init=/nix/store/x8lvlh5c5rdfaf25w50fpdylkcwd3ihy-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/c1.large.arm": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/c1.large.arm
kernel ${base-url}/kernel init=/nix/store/gbhizlyjj5gc3fayvw79dsii6ac5yb74-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd cma=0M biosdevname=0 net.ifnames=0 console=ttyAMA0,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/s1.large.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/s1.large.x86
kernel ${base-url}/kernel init=/nix/store/nnic31dppmzamq7l3sn3iyjks55qsrn5-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/x1.small.x86": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_18_03/x1.small.x86
kernel ${base-url}/kernel init=/nix/store/bd42lgd9rmz4xmq3zgs8j31rf0g7fn4q-nixos-system-install-environment-18.03.132610.49a6964a425/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
	"18_03/xx.nano.s390": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet
shell
`,
	"20_09/c3.small.x86:nix-store-path-masquerading-as-version": `echo Packet.net Baremetal - iPXE boot
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell/nixos/nixos_20_09/nix-store-path-masquerading-as-version
kernel ${base-url}/kernel init=/nix/store/nix-store-path-masquerading-as-version/init initrd=initrd console=ttyS1,115200 loglevel=7
initrd ${base-url}/initrd
boot
`,
}
