package rancher

import (
	"github.com/packethost/boots/ipxe"
	"github.com/packethost/boots/job"
)

func init() {
	job.RegisterDistro("rancher", bootScript)
}

func bootScript(j job.Job, s *ipxe.Script) {
	s.PhoneHome("provisioning.104.01")
	s.Set("base-url", "http://releases.rancher.com/os/packet")
	s.Kernel("${base-url}/vmlinuz")

	kernelParams(j, s)

	s.Initrd("${base-url}/initrd")
	s.Boot()
}

func kernelParams(j job.Job, s *ipxe.Script) {
	s.Args("console=ttyS1,115200n8")
	s.Args("rancher.cloud_init.datasources=[url:${base-url}/packet.sh]")

	switch j.PlanSlug() {
	case "baremetal_0", "baremetal_1", "t1.small.x86", "c1.small.x86":
		s.Args("rancher.network.interfaces.eth0.dhcp=true")
		s.Args("rancher.network.interfaces.eth2.dhcp=true")
	}

	s.Args("tinkerbell=${tinkerbell}")
}
