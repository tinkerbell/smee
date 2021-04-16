package vmware

import (
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
	for typ, script := range type2pxe {
		t.Run(typ, func(t *testing.T) {
			for version, bootScript := range versions {
				t.Run(version, func(t *testing.T) {

					m := job.NewMock(t, typ, facility)
					m.SetMAC("00:00:ba:dd:be:ef")

					s := ipxe.Script{}
					bootScript(m.Job(), &s)
					got := string(s.Bytes())

					want := fmt.Sprintf(script, version)
					if !strings.Contains(want, version) {
						t.Fatalf("expected %s to be present in script:%v", version, want)
					}
					if want != got {
						t.Fatalf("%s bad iPXE script:\n%v", typ, diff.LineDiff(want, got))
					}
				})
			}
		})
	}
}

var type2pxe = map[string]string{
	"baremetal_0": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	"baremetal_1": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	"baremetal_2": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	"baremetal_3": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	"baremetal_s": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/mboot.c32 -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
	"c2.medium.x86": `dhcp

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set base-url http://install.ewr1.packet.net/vmware/%s
kernel ${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg ks=${tinkerbell}/vmware/ks-esxi.cfg netdevice=00:00:ba:dd:be:ef ksdevice=00:00:ba:dd:be:ef
boot
`,
}

var versions = map[string]func(job.Job, *ipxe.Script){
	"esxi-5.5.0.update03": bootScriptVmwareEsxi55,
	"esxi-6.0.0.update03": bootScriptVmwareEsxi60,
	"esxi-6.5.0":          bootScriptVmwareEsxi65,
}
