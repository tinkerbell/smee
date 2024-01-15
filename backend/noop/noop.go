// Package noop is a backend handler that does nothing.
package noop

import (
	"context"
	"errors"
	"net"

	"github.com/tinkerbell/smee/dhcp/data"
)

// Handler is a noop backend.
type Handler struct{}

// GetByMac returns an error.
func (h Handler) GetByMac(_ context.Context, _ net.HardwareAddr) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("no backend specified, please specify a backend")
}

// GetByIP returns an error.
func (h Handler) GetByIP(_ context.Context, _ net.IP) (*data.DHCP, *data.Netboot, error) {
	return nil, nil, errors.New("no backend specified, please specify a backend")
}
