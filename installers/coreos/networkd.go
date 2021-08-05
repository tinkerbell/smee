package coreos

// TODO(SWE-338) have coreos register http handler for /installers/coreos and move this into coreos package

import (
	"net"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/files/unit"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/packet"
)

const bondName = "bond0"

var (
	defaultIPv4 = formatCIDR(net.IPv4zero, net.CIDRMask(0, 32))
	defaultIPv6 = formatCIDR(net.IPv6zero, net.CIDRMask(0, 128))
	privateIPv4 = formatCIDR(net.IPv4(10, 0, 0, 0), net.CIDRMask(8, 32))
)

var bondOptions = []string{
	"TransmitHashPolicy=layer3+4",
	"MIIMonitorSec=.1",
}

func configureBondSlaveUnit(j job.Job, u *unit.Unit, p packet.Port) bool {
	if p.Data.Bond != bondName {
		return false
	}

	u.AddSection("Match").Add("MACAddress", p.Data.MAC.String())
	u.AddSection("Network", "Bond="+bondName)

	return true
}

func configureBondDevUnit(j job.Job, u *unit.Unit) {
	u.AddSection("NetDev", "Name="+bondName, "Kind=bond").Add("MACAddress", j.InterfaceMAC(0).String())

	s := u.AddSection("Bond", bondOptions...)
	switch int(j.BondingMode()) {
	case 4: // LACP
		s.Add("Mode", "802.3ad")
		s.Add("LACPTransmitRate", "fast")
	case 5: // TLB
		s.Add("Mode", "balance-tlb")
	}
}

func configureNetworkUnit(j job.Job, u *unit.Unit) {
	u.AddSection("Match", "Name="+bondName)
	s := u.AddSection("Network")

	for _, ip := range conf.DNSServers {
		s.Add("DNS", ip.String())
	}

	for _, ip := range j.InstanceIPs() {
		s.Add("Address", formatCIDR(ip.Address, net.IPMask(ip.Netmask)))

		if !ip.Management {
			continue // TODO: Confirm this is the correct behavior.
		}

		var dest string
		switch ip.Family {
		case 4:
			if ip.Public {
				dest = defaultIPv4
			} else {
				dest = privateIPv4
			}
		case 6:
			dest = defaultIPv6
		}
		if dest != "" {
			u.AddSection("Route").Add("Destination", dest).Add("Gateway", ip.Gateway.String())
		}
	}
}

func formatCIDR(ip net.IP, mask net.IPMask) string {
	n := net.IPNet{
		IP:   ip,
		Mask: mask,
	}

	return n.String()
}
