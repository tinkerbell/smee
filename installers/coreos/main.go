package coreos

import (
	"context"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

const (
	IgnitionPathCoreos  = "/coreos/ignition.json"
	IgnitionPathFlatcar = "/flatcar/ignition.json"
	OEMPath             = "/coreos/oem.tgz"
)

// Alternative Base URLs
// http://storage.googleapis.com/alpha.release.core-os.net/amd64-usr/current
// http://storage.googleapis.com/users.developer.core-os.net/mischief/boards/amd64-usr/962.0.0+2016-02-23-2254

type Installer struct{}

func (i Installer) BootScript() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.PhoneHome("provisioning.104.01")
		s.Set("base-url", conf.MirrorURL)
		s.Kernel("${base-url}/" + kernelPath(j))

		ks := kernelParams(j, s)

		ks.Initrd("${base-url}/" + initrdPath(j))
		ks.Boot()

		return ks
	}
}

func kernelParams(j job.Job, s ipxe.Script) ipxe.Script {
	distro := j.OperatingSystem().Distro

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
	s.Args(distro + ".autologin")
	s.Args(distro + ".first_boot=1")

	// Ignition
	s.Args(distro + ".config.url=${tinkerbell}/" + distro + "/ignition.json")

	// Environment Variables
	s.Args("systemd.setenv=oem_url=${tinkerbell}/" + distro + "/oem.tgz") // To replace the files in our included OEM.
	s.Args("systemd.setenv=phone_home_url=${tinkerbell}/phone-home")

	return s
}

func kernelPath(j job.Job) string {
	distro := j.OperatingSystem().Distro
	if j.IsARM() {
		return distro + "-arm.vmlinuz"
	}

	return distro + "_production_pxe.vmlinuz"
}

func initrdPath(j job.Job) string {
	distro := j.OperatingSystem().Distro
	if j.IsARM() {
		return distro + "-arm.cpio.gz"
	}

	return distro + "_production_pxe_image.cpio.gz"
}
