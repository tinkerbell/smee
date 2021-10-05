package custom_ipxe

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
	for typ, script := range type2Script {
		t.Run(typ, func(t *testing.T) {
			m := job.NewMock(t, typ, facility)
			m.SetIPXEScriptURL("http://127.0.0.1/fake_ipxe_url")

			s := ipxe.Script{}
			s.Echo("Tinkerbell Boots iPXE")
			s.Set("iface", "eth0").Or("shell")
			s.Set("tinkerbell", "http://127.0.0.1")
			s.Set("ipxe_cloud_config", "packet")
			ci := Installer{}
			bs := ci.BootScript()(context.Background(), m.Job(), s)
			got := string(bs.Bytes())
			if script != got {
				t.Fatalf("%s bad iPXE script:\n%v", typ, diff.LineDiff(script, got))
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

set packet_facility ` + facility + `
set packet_plan baremetal_0
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_1
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_2
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_3
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_2a
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_2a2
chain --autofree http://127.0.0.1/fake_ipxe_url
`,
	"baremetal_2a4": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set packet_facility ` + facility + `
set packet_plan baremetal_2a4
chain --autofree http://127.0.0.1/fake_ipxe_url
`,
	"baremetal_2a5": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set packet_facility ` + facility + `
set packet_plan baremetal_2a5
chain --autofree http://127.0.0.1/fake_ipxe_url
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

set packet_facility ` + facility + `
set packet_plan baremetal_hua
chain --autofree http://127.0.0.1/fake_ipxe_url
`,
	"c2.large.arm": `echo Tinkerbell Boots iPXE
set iface eth0 || shell
set tinkerbell http://127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch ${tinkerbell}/phone-home##params
imgfree

set packet_facility ` + facility + `
set packet_plan c2.large.arm
chain --autofree http://127.0.0.1/fake_ipxe_url
`,
}
