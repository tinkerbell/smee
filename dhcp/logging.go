package dhcp

import (
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/packethost/pkg/log"
)

var dhcplog log.Logger

func Init(l log.Logger) {
	dhcplog = l.Package("dhcp")
	dhcp4.Init(dhcplog)
}
