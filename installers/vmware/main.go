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

func (i Installer) BootScriptDefault() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.Shell()

		j.DisablePXE(ctx)
		j.MarkDeviceActive(ctx)

		return s
	}
}

func (i Installer) BootScriptVmwareEsxi55() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-5.5.0.update03")
	}
}

func (i Installer) BootScriptVmwareEsxi60() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-6.0.0.update03")
	}
}

func (i Installer) BootScriptVmwareEsxi65() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-6.5.0")
	}
}

func (i Installer) BootScriptVmwareEsxi67() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-6.7.0")
	}
}

func (i Installer) BootScriptVmwareEsxi70() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-7.0.0")
	}
}

func (i Installer) BootScriptVmwareEsxi70U2a() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		return script(j, s, "/vmware/esxi-7.0U2a")
	}
}

func script(j job.Job, s ipxe.Script, basePath string) ipxe.Script {
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
