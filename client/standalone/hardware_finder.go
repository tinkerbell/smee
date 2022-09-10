package standalone

import (
	"context"
	"encoding/json"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
)

// HardwareFinder is a type for statically looking up hardware.
type HardwareFinder struct {
	db []*DiscoverStandalone
}

// NewHardwareFinder returns a Finder given a JSON file that is formatted as a slice of
// DiscoverStandalone.
func NewHardwareFinder(path string) (*HardwareFinder, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %q", path)
	}
	db := []*DiscoverStandalone{}
	err = json.Unmarshal(content, &db)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse configuration file %q", path)
	}

	return &HardwareFinder{
		db: db,
	}, nil
}

// ByIP returns a Discoverer for a particular IP.
func (f *HardwareFinder) ByIP(_ context.Context, ip net.IP) (client.Discoverer, error) {
	for _, d := range f.db {
		for _, hip := range d.HardwareIPs() {
			if hip.Address.Equal(ip) {
				return d, nil
			}
		}
	}

	return nil, errors.Errorf("no hardware found for ip %q", ip)
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *HardwareFinder) ByMAC(_ context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	for _, d := range f.db {
		if d.MAC().String() == mac.String() {
			return d, nil
		}
	}

	return nil, errors.Errorf("no entry for MAC %q in standalone data", mac.String())
}
