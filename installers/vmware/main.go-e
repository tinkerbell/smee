package vmware

import (
	"github.com/packethost/boots/env"
	"github.com/packethost/boots/ipxe"
	"github.com/packethost/boots/job"
)

func init() {
	job.RegisterSlug("vmware_esxi_5_5", bootScriptVmwareEsxi55)
	job.RegisterSlug("vmware_esxi_6_0", bootScriptVmwareEsxi60)
	job.RegisterSlug("vmware_esxi_6_5", bootScriptVmwareEsxi65)
	job.RegisterSlug("vmware_esxi_6_7", bootScriptVmwareEsxi67)
	job.RegisterDistro("vmware", bootScriptDefault)
}

func bootScriptDefault(j job.Job, s *ipxe.Script) {
	s.Shell()

	// We don't need to actually provision anything
	j.DisablePXE()
	j.MarkDeviceActive()
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

func bootScriptVmwareEsxi(j job.Job, s *ipxe.Script, basePath string) {
	s.DHCP()
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", env.MirrorBaseUrl+basePath)
	if j.IsUEFI() {
		s.Kernel("${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg")
	} else {
		s.Kernel("${base-url}/mboot.c32 -c ${base-url}/boot.cfg")
	}

	kernelParams(j, s, "/vmware/ks-esxi.cfg")

	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script, kickstartPath string) {
	s.Args("ks=${boots}" + kickstartPath)

	vmnic := j.PrimaryNIC().String()
	s.Args("netdevice=" + vmnic)
	s.Args("ksdevice=" + vmnic)
}
