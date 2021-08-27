package vmware

import (
	"context"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

func init() {
	job.RegisterSlug("vmware_esxi_5_5", bootScriptVmwareEsxi55)
	job.RegisterSlug("vmware_esxi_6_0", bootScriptVmwareEsxi60)
	job.RegisterSlug("vmware_esxi_6_5", bootScriptVmwareEsxi65)
	job.RegisterSlug("vmware_esxi_6_7", bootScriptVmwareEsxi67)
	job.RegisterSlug("vmware_esxi_7_0", bootScriptVmwareEsxi70)
	job.RegisterSlug("vmware_esxi_6_5_vcf", bootScriptVmwareEsxi65)
	job.RegisterSlug("vmware_esxi_6_7_vcf", bootScriptVmwareEsxi67)
	job.RegisterSlug("vmware_esxi_7_0_vcf", bootScriptVmwareEsxi70)
	job.RegisterDistro("vmware", bootScriptDefault)
}

func bootScriptDefault(j job.Job, s *ipxe.Script) {
	s.Shell()

	// We don't need to actually provision anything
	// TODO(@tobert) passing context through to here would mean changing the
	// signature for all installer functions and this is the only site that
	// needs it, so these will not have trace context
	j.DisablePXE(context.Background())
	j.MarkDeviceActive(context.Background())
}

func bootScriptVmwareEsxi55(j job.Job, s *ipxe.Script) {
	bootScriptVmwareEsxi(j, s, "/vmware/esxi-5.5.0.update03")
}

func bootScriptVmwareEsxi60(j job.Job, s *ipxe.Script) {
	bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.0.0.update03")
}

func bootScriptVmwareEsxi65(j job.Job, s *ipxe.Script) {
	bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.5.0")
}

func bootScriptVmwareEsxi67(j job.Job, s *ipxe.Script) {
	bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.7.0")
}

func bootScriptVmwareEsxi70(j job.Job, s *ipxe.Script) {
	bootScriptVmwareEsxi(j, s, "/vmware/esxi-7.0.0")
}

func bootScriptVmwareEsxi(j job.Job, s *ipxe.Script, basePath string) {
	s.DHCP()
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", conf.MirrorBaseUrl+basePath)
	if j.IsUEFI() {
		s.Kernel("${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg")
	} else {
		s.Kernel("${base-url}/mboot.c32 -c ${base-url}/boot.cfg")
	}

	kernelParams(j, s, "/vmware/ks-esxi.cfg")

	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script, kickstartPath string) {
	s.Args("ks=${tinkerbell}" + kickstartPath)

	vmnic := j.PrimaryNIC().String()
	s.Args("netdevice=" + vmnic)
	s.Args("ksdevice=" + vmnic)
}
