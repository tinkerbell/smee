package flatcar

import (
	"context"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

const (
	IgnitionPathFlatcar = "/flatcar/ignition.json"
)

// Alternative Base URLs
// http://storage.googleapis.com/alpha.release.core-os.net/amd64-usr/current
// http://storage.googleapis.com/users.developer.core-os.net/mischief/boards/amd64-usr/962.0.0+2016-02-23-2254

type Installer struct{}

func (i Installer) BootScript() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.PhoneHome("provisioning.104.01")
		s.Set("base-url", conf.MirrorBaseURL+"/misc/tinkerbell")
		s.Kernel("${base-url}/" + kernelPath(j))

		ks := kernelParams(j, s)

		ks.Initrd("${base-url}/" + initrdPath(j))
		ks.Boot()

		return ks
	}
}

func kernelParams(j job.Job, s ipxe.Script) ipxe.Script {
	// Linux Kernel
	if j.IsARM() {
		s.Args("console=ttyAMA0,115200")
		s.Args("initrd=" + initrdPath(j))
	} else {
		s.Args("console=ttyS1,115200n8 console=tty0 vga=773")
		s.Args("initrd=" + initrdPath(j))
	}

	s.Args("bonding.max_bonds=0") // To prevent the wrong bond from coming up before our configs are in place.

	// CoreOS
	s.Args("flatcar.autologin")
	s.Args("flatcar.first_boot=1")

	// Ignition
	s.Args("flatcar.config.url=${tinkerbell}/flatcar/ignition.json")

	// Environment Variables
	s.Args("systemd.setenv=phone_home_url=${tinkerbell}/phone-home")

	return s
}

func kernelPath(j job.Job) string {
	if j.IsARM() {
		return "flatcar-arm.vmlinuz"
	}

	return "flatcar_production_pxe.vmlinuz"
}

func initrdPath(j job.Job) string {
	if j.IsARM() {
		return "flatcar-arm.cpio.gz"
	}

	return "flatcar_production_pxe_image.cpio.gz"
}
