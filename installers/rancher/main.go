package rancher

import (
	"context"

	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

type Installer struct{}

func (i Installer) BootScript() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.PhoneHome("provisioning.104.01")
		s.Set("base-url", "http://releases.rancher.com/os/packet")
		s.Kernel("${base-url}/vmlinuz")

		ks := kernelParams(j, s)

		ks.Initrd("${base-url}/initrd")
		ks.Boot()

		return ks
	}
}

func kernelParams(j job.Job, s ipxe.Script) ipxe.Script {
	s.Args("console=ttyS1,115200n8")
	s.Args("rancher.cloud_init.datasources=[url:${base-url}/packet.sh]")

	switch j.PlanSlug() {
	case "baremetal_0", "baremetal_1", "t1.small.x86", "c1.small.x86":
		s.Args("rancher.network.interfaces.eth0.dhcp=true")
		s.Args("rancher.network.interfaces.eth2.dhcp=true")
	}

	s.Args("tinkerbell=${tinkerbell}")

	return s
}
