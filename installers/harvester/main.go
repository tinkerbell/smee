package harvester

import (
	"context"
	"fmt"

	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

const (
	defaultVersion = "v1.0.2"
)

type installer struct{}

func Installer() job.BootScripter {
	return installer{}
}

func (i installer) BootScript(string) job.BootScript {
	return func(ctx context.Context, j job.Job, s *ipxe.Script) {
		// disable subsequent ipxe
		defer j.DisablePXE(ctx)
		// broken up logic for easier tests.
		// mock job triggers failures when DisablePXE is invoked
		// generate kernel params
		generateBootScript(ctx, j, s)
	}

}

func generateBootScript(ctx context.Context, j job.Job, s *ipxe.Script) {
	s.PhoneHome("provisioning.104.01")
	if len(j.OSIEBaseURL()) != 0 {
		s.Set("base-url", j.OSIEBaseURL())
	} else {
		s.Set("base-url", "https://releases.rancher.com/harvester")
	}
	j.With("parsed userdata", j.UserData())

	version := defaultVersion
	if j.OperatingSystem().Version != "" {
		version = j.OperatingSystem().Version
	}
	kernelParams(j, s, version)
	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script, version string) {

	s.Kernel(fmt.Sprintf("${base-url}/%s/harvester-%s-vmlinuz-amd64", version, version))
	s.Args("rd.cos.disable", "rd.noverifyssl", "net.ifnames=1", "console=tty1", "harvester.install.automatic=true", "boot_cmd=\"echo include_ping_test=yes >> /etc/conf.d/net-online\"")
	s.Args(fmt.Sprintf("root=live:${base-url}/%s/harvester-%s-rootfs-amd64.squashfs", version, version))
	if len(j.UserData()) != 0 {
		s.Args(j.UserData())
	}
	s.Initrd(fmt.Sprintf("${base-url}/%s/harvester-%s-initrd-amd64", version, version))
}
