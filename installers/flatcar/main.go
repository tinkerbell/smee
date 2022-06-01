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

type installer struct{}

func Installer() job.BootScripter {
	return installer{}
}

func (i installer) BootScript(string) job.BootScript {
	return bootScript
}

func bootScript(ctx context.Context, j job.Job, s *ipxe.Script) {
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", conf.OsieVendorServicesURL+"/flatcar")
	s.Kernel("${base-url}/" + kernelPath(j))

	kernelParams(j, s)

	s.Initrd("${base-url}/" + initrdPath(j))
	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script) {
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
