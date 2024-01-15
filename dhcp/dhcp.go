// Package dhcp providers UDP listening and serving functionality.
package dhcp

import (
	"context"
	"net"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/tinkerbell/smee/dhcp/data"
	"golang.org/x/net/ipv4"
)

// Handler is a type that defines the handler function to be called every time a
// valid DHCPv4 message is received
// type Handler func(ctx context.Context, conn net.PacketConn, d data.Packet).
type Handler interface {
	Handle(ctx context.Context, conn *ipv4.PacketConn, d data.Packet)
}

// Server represents a DHCPv4 server object.
type Server struct {
	Conn     net.PacketConn
	Handlers []Handler
	Logger   logr.Logger
}

// Serve serves requests.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	s.Logger.Info("Server listening on", "addr", s.Conn.LocalAddr())

	nConn := ipv4.NewPacketConn(s.Conn)
	if err := nConn.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		s.Logger.Info("error setting control message", "err", err)
		return err
	}

	defer func() {
		_ = nConn.Close()
	}()
	for {
		// Max UDP packet size is 65535. Max DHCPv4 packet size is 576. An ethernet frame is 1500 bytes.
		// We use 4096 as a reasonable buffer size. dhcpv4.FromBytes will handle the rest.
		rbuf := make([]byte, 4096)
		n, cm, peer, err := nConn.ReadFrom(rbuf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			s.Logger.Info("error reading from packet conn", "err", err)
			return err
		}

		m, err := dhcpv4.FromBytes(rbuf[:n])
		if err != nil {
			s.Logger.Info("error parsing DHCPv4 request", "err", err)
			continue
		}

		upeer, ok := peer.(*net.UDPAddr)
		if !ok {
			s.Logger.Info("not a UDP connection? Peer is", "peer", peer)
			continue
		}
		// Set peer to broadcast if the client did not have an IP.
		if upeer.IP == nil || upeer.IP.To4().Equal(net.IPv4zero) {
			upeer = &net.UDPAddr{
				IP:   net.IPv4bcast,
				Port: upeer.Port,
			}
		}

		var ifName string
		if n, err := net.InterfaceByIndex(cm.IfIndex); err == nil {
			ifName = n.Name
		}

		for _, handler := range s.Handlers {
			go handler.Handle(ctx, nConn, data.Packet{Peer: upeer, Pkt: m, Md: &data.Metadata{IfName: ifName, IfIndex: cm.IfIndex}})
		}
	}
}

// Close sends a termination request to the server, and closes the UDP listener.
func (s *Server) Close() error {
	return s.Conn.Close()
}

// NewServer initializes and returns a new Server object.
func NewServer(ifname string, addr *net.UDPAddr, handler ...Handler) (*Server, error) {
	s := &Server{
		Handlers: handler,
		Logger:   logr.Discard(),
	}

	if s.Conn == nil {
		var err error
		conn, err := server4.NewIPv4UDPConn(ifname, addr)
		if err != nil {
			return nil, err
		}
		s.Conn = conn
	}
	return s, nil
}
