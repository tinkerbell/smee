package vmware

import (
	"context"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

const (
	KickstartPath = "/vmware/ks-esxi.cfg"
)

type installer struct {
	extraIPXEVars [][]string
}

// pass var here
func Installer(dynamicIPXEVars [][]string) job.BootScripter {
	i := installer{
		extraIPXEVars: dynamicIPXEVars,
	}

	return i
}

var slug2Paths = map[string]string{
	"vmware_esxi_5_5":     "esxi-5.5.0.update03",
	"vmware_esxi_6_0":     "esxi-6.0.0.update03",
	"vmware_esxi_6_5":     "esxi-6.5.0",
	"vmware_esxi_6_5_vcf": "esxi-6.5.0",
	"vmware_esxi_6_7":     "esxi-6.7.0",
	"vmware_esxi_6_7_vcf": "esxi-6.7.0",
	"vmware_esxi_7_0U2a":  "esxi-7.0U2a",
	"vmware_esxi_7_0":     "esxi-7.0.0",
	"vmware_esxi_7_0_vcf": "esxi-7.0.0",
	"vmware":              "abort",
}

func (i installer) BootScript(slug string) job.BootScript {
	path := slug2Paths[slug]
	if path == "" {
		panic("unknown slug:" + slug)
	}
	if path == "abort" {
		return func(ctx context.Context, j job.Job, s *ipxe.Script) {
			s.Shell()

			j.DisablePXE(ctx)
			j.MarkDeviceActive(ctx)
		}
	}

	return func(ctx context.Context, j job.Job, s *ipxe.Script) {
		script(i, j, s, path)
	}
}

func script(i installer, j job.Job, s *ipxe.Script, basePath string) {
	for _, kv := range i.extraIPXEVars {
		s.Set(kv[0], kv[1])
	}

	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", conf.OsieVendorServicesURL+"/vmware/"+basePath)
	if j.IsUEFI() {
		s.Kernel("${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg")
	} else {
		s.Kernel("${base-url}/mboot.c32 -c ${base-url}/boot.cfg")
	}

	kernelParams(j, s)
	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script) {
	s.Args("ks=${tinkerbell}/vmware/ks-esxi.cfg")

	vmnic := j.PrimaryNIC().String()
	s.Args("netdevice=" + vmnic)
	s.Args("ksdevice=" + vmnic)
}
