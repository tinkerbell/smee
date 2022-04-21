package vmware

import (
	"context"
	"fmt"
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

func TestScriptPerType(t *testing.T) {
	for mode, planAndScript := range pxeByPlan {
		t.Run(mode, func(t *testing.T) {
			for slug, path := range slug2Paths {
				t.Run(slug, func(t *testing.T) {
					if slug == "vmware" {
						t.Skipf("skipping vmware slug/default because it panics when calling j.DisablePXE() because client is nil and mocking client is not worth it imo")
					}
					plan := planAndScript.plan
					script := planAndScript.script

					m := job.NewMock(t, plan, facility)
					m.SetMAC("00:00:ba:dd:be:ef")

					s := ipxe.NewScript()
					s.Set("iface", "eth0")
					s.Or("shell")
					s.Set("tinkerbell", "http://127.0.0.1")
					s.Set("syslog_host", "127.0.0.1")
					s.Set("ipxe_cloud_config", "packet")

					Installer().BootScript(slug)(context.Background(), m.Job(), s)
					got := string(s.Bytes())

					want := fmt.Sprintf(script, path)
					if !strings.Contains(want, path) {
						t.Fatalf("expected %s to be present in script:%v", path, want)
					}
					if want != got {
						t.Fatalf("bad iPXE script:\n%v", diff.LineDiff(want, got))
					}
				})
			}
		})
	}
}

var pxeByPlan = map[string]struct {
	plan   string
	script string
}{
	"bios": {
		plan: "c3.small.x86",
		script: `#!ipxe

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

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`},
	"uefi": {
		plan: "c2.medium.x86",
		script: `#!ipxe

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

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	},
}
