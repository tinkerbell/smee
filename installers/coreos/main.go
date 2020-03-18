package coreos

import (
	"github.com/packethost/boots/env"
	"github.com/packethost/boots/ipxe"
	"github.com/packethost/boots/job"
)

func init() {
	job.RegisterDistro("coreos", bootScript)
	job.RegisterDistro("flatcar", bootScript)
}

// Alternative Base URLs
// http://storage.googleapis.com/alpha.release.core-os.net/amd64-usr/current
// http://storage.googleapis.com/users.developer.core-os.net/mischief/boards/amd64-usr/962.0.0+2016-02-23-2254

func bootScript(j job.Job, s *ipxe.Script) {
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", env.MirrorURL)
	s.Kernel("${base-url}/" + kernelPath(j))

	kernelParams(j, s)

	s.Initrd("${base-url}/" + initrdPath(j))
	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script) {
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
	s.Args(distro + ".config.url=${boots}/" + distro + "/ignition.json")

	// Environment Variables
	s.Args("systemd.setenv=oem_url=${boots}/" + distro + "/oem.tgz") // To replace the files in our included OEM.
	s.Args("systemd.setenv=phone_home_url=${boots}/phone-home")
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
