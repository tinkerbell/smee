package job

import (
	"context"
	"net"

	"github.com/golang/groupcache/singleflight"
	"github.com/tinkerbell/boots/packet"
)

var (
	servers singleflight.Group
)

func discoverHardwareFromDHCP(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (packet.Discovery, error) {
	fetch := func() (interface{}, error) {
		return client.DiscoverHardwareFromDHCP(ctx, mac, giaddr, circuitID)
	}
	v, err := servers.Do(mac.String(), fetch)
	if err != nil {
		return nil, err
	}

	return v.(packet.Discovery), nil
}

func discoverHardwareFromIP(ctx context.Context, ip net.IP) (packet.Discovery, error) {
	fetch := func() (interface{}, error) {
		return client.DiscoverHardwareFromIP(ctx, ip)
	}
	v, err := servers.Do(ip.String(), fetch)
	if err != nil {
		return nil, err
	}

	return v.(packet.Discovery), nil
}
