package flatcar

import (
	"context"
	"os"
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
	for arch, tt := range pxeByPlan {
		t.Run(arch, func(t *testing.T) {
			m := job.NewMock(t, tt.plan, facility)
			m.SetOSDistro("flatcar")

			s := ipxe.Script{}
			s.Echo("Tinkerbell Boots iPXE")
			s.Set("iface", "eth0").Or("shell")
			s.Set("tinkerbell", "http://127.0.0.1")
			s.Set("ipxe_cloud_config", "packet")
			i := Installer{}
			bs := i.BootScript()(context.Background(), m.Job(), s)
			got := string(bs.Bytes())
			if tt.script != got {
				t.Fatalf("bad iPXE script:\n%v", diff.LineDiff(tt.script, got))
			}
		})
	}
}

var pxeByPlan = map[string]struct {
	plan   string
	script string
}{
	"x86_64": {
		"c3.small.x86",
		`echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/flatcar_production_pxe.vmlinuz console=ttyS1,115200n8 console=tty0 vga=773 initrd=flatcar_production_pxe_image.cpio.gz bonding.max_bonds=0 flatcar.autologin flatcar.first_boot=1 flatcar.config.url=${tinkerbell}/flatcar/ignition.json systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/flatcar_production_pxe_image.cpio.gz
boot
`},
	"aarch64": {
		"c3.large.arm",
		`echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.` + facility + `.packet.net/misc/tinkerbell
kernel ${base-url}/flatcar-arm.vmlinuz console=ttyAMA0,115200 initrd=flatcar-arm.cpio.gz bonding.max_bonds=0 flatcar.autologin flatcar.first_boot=1 flatcar.config.url=${tinkerbell}/flatcar/ignition.json systemd.setenv=phone_home_url=${tinkerbell}/phone-home
initrd ${base-url}/flatcar-arm.cpio.gz
boot
`},
}
