// Package noop is a handler that does nothing.
package noop

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/tinkerbell/smee/dhcp/data"
)

// Handler is a noop handler.
type Handler struct {
	Log logr.Logger
}

// Handle is the noop handler function.
func (n *Handler) Handle(_ context.Context, _ net.PacketConn, _ data.Packet) {
	msg := "no handler specified. please specify a handler"
	if n.Log.GetSink() == nil {
		stdr.New(log.New(os.Stdout, "", log.Lshortfile)).Info(msg)
	} else {
		n.Log.Info(msg)
	}
}
