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
			for slug, tt := range inputsBySlug {
				t.Run(slug, func(t *testing.T) {
					plan := planAndScript.plan
					script := planAndScript.script

					m := job.NewMock(t, plan, facility)
					m.SetMAC("00:00:ba:dd:be:ef")

					s := ipxe.Script{}
					bs := tt.script(context.Background(), m.Job(), s)
					got := string(bs.Bytes())

					want := fmt.Sprintf(script, tt.path)
					if !strings.Contains(want, tt.path) {
						t.Fatalf("expected %s to be present in script:%v", tt.path, want)
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
		script: `
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
		script: `
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

var inputsBySlug = map[string]struct {
	path   string
	script job.BootScript
}{
	"vmware_esxi_5_5":     {"esxi-5.5.0.update03", Installer{}.BootScriptVmwareEsxi55()},
	"vmware_esxi_6_0":     {"esxi-6.0.0.update03", Installer{}.BootScriptVmwareEsxi60()},
	"vmware_esxi_6_5":     {"esxi-6.5.0", Installer{}.BootScriptVmwareEsxi65()},
	"vmware_esxi_6_7":     {"esxi-6.7.0", Installer{}.BootScriptVmwareEsxi67()},
	"vmware_esxi_7_0":     {"esxi-7.0.0", Installer{}.BootScriptVmwareEsxi70()},
	"vmware_esxi_7_0U2a":  {"esxi-7.0U2a", Installer{}.BootScriptVmwareEsxi70U2a()},
	"vmware_esxi_6_5_vcf": {"esxi-6.5.0", Installer{}.BootScriptVmwareEsxi65()},
	"vmware_esxi_6_7_vcf": {"esxi-6.7.0", Installer{}.BootScriptVmwareEsxi67()},
	"vmware_esxi_7_0_vcf": {"esxi-7.0.0", Installer{}.BootScriptVmwareEsxi70()},
	// "vmware":              {path: "", script: Installer{}.BootScriptDefault()}, panics because client is nil and mocking client is not worth it imo
}
