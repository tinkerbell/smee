package noop

import (
	"context"
	"errors"
	"net"

	"github.com/tinkerbell/smee/internal/dhcp/data"
)

type Backend struct{}

func (n Backend) GetByMac(context.Context, net.HardwareAddr) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("noop backend always returns an error")
}

func (n Backend) GetByIP(context.Context, net.IP) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("noop backend always returns an error")
}
