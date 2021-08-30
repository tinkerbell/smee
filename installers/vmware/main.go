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

type Installer struct{}

func (i Installer) BootScriptDefault() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		s.Shell()

		// We don't need to actually provision anything
		// TODO(@tobert) passing context through to here would mean changing the
		// signature for all installer functions and this is the only site that
		// needs it, so these will not have trace context
		j.DisablePXE(context.Background())
		j.MarkDeviceActive(context.Background())

		return s
	}
}

func (i Installer) BootScriptVmwareEsxi55() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		return bootScriptVmwareEsxi(j, s, "/vmware/esxi-5.5.0.update03")
	}
}

func (i Installer) BootScriptVmwareEsxi60() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		return bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.0.0.update03")
	}
}

func (i Installer) BootScriptVmwareEsxi65() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		return bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.5.0")
	}
}

func (i Installer) BootScriptVmwareEsxi67() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		return bootScriptVmwareEsxi(j, s, "/vmware/esxi-6.7.0")
	}
}

func (i Installer) BootScriptVmwareEsxi70() func(j job.Job, s ipxe.Script) ipxe.Script {
	return func(j job.Job, s ipxe.Script) ipxe.Script {
		return bootScriptVmwareEsxi(j, s, "/vmware/esxi-7.0.0")
	}
}

func bootScriptVmwareEsxi(j job.Job, s ipxe.Script, basePath string) ipxe.Script {
	s.DHCP()
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", conf.MirrorBaseUrl+basePath)
	if j.IsUEFI() {
		s.Kernel("${base-url}/efi/boot/bootx64.efi -c ${base-url}/boot.cfg")
	} else {
		s.Kernel("${base-url}/mboot.c32 -c ${base-url}/boot.cfg")
	}

	ks := kernelParams(j, s, "/vmware/ks-esxi.cfg")

	ks.Boot()

	return ks
}

func kernelParams(j job.Job, s ipxe.Script, kickstartPath string) ipxe.Script {
	s.Args("ks=${tinkerbell}" + kickstartPath)

	vmnic := j.PrimaryNIC().String()
	s.Args("netdevice=" + vmnic)
	s.Args("ksdevice=" + vmnic)

	return s
}
