package cacher

import (
	"context"
	"encoding/json"
	"net"

	cacherClient "github.com/packethost/cacher/client"
	"github.com/packethost/cacher/protos/cacher"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
)

// HardwareFinder is a type that can discover hardware from a cacher client.
type HardwareFinder struct {
	cc cacher.CacherClient
}

// NewHardwareFinder returns a github.com/packethost/cacher/client Finder.
func NewHardwareFinder(facility string) (*HardwareFinder, error) {
	cc, err := cacherClient.New(facility)
	if err != nil {
		return nil, errors.Wrap(err, "connect to cacher")
	}

	return &HardwareFinder{cc}, nil
}

// ByIP returns a Discoverer for a particular IP.
func (f *HardwareFinder) ByIP(ctx context.Context, ip net.IP) (client.Discoverer, error) {
	resp, err := f.cc.ByIP(ctx, &cacher.GetRequest{
		IP: ip.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by ip from cacher")
	}
	if len(resp.JSON) == 0 {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryCacher{}
	err = json.Unmarshal([]byte(resp.JSON), d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *HardwareFinder) ByMAC(ctx context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	resp, err := f.cc.ByMAC(ctx, &cacher.GetRequest{
		MAC: mac.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by mac from cacher")
	}
	if len(resp.JSON) == 0 {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryCacher{}
	err = json.Unmarshal([]byte(resp.JSON), d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}
