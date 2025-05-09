package server

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
	"github.com/mvellasco/smee/internal/dhcp/data"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/nettest"
)

type mock struct {
	Log         logr.Logger
	ServerIP    net.IP
	LeaseTime   uint32
	YourIP      net.IP
	NameServers []net.IP
	SubnetMask  net.IPMask
	Router      net.IP
}

func (m *mock) Handle(_ context.Context, conn *ipv4.PacketConn, d data.Packet) {
	if m.Log.GetSink() == nil {
		m.Log = logr.Discard()
	}

	mods := m.setOpts()
	switch mt := d.Pkt.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		mods = append(mods, dhcpv4.WithMessageType(dhcpv4.MessageTypeOffer))
	case dhcpv4.MessageTypeRequest:
		mods = append(mods, dhcpv4.WithMessageType(dhcpv4.MessageTypeAck))
	case dhcpv4.MessageTypeRelease:
		mods = append(mods, dhcpv4.WithMessageType(dhcpv4.MessageTypeAck))
	default:
		m.Log.Info("unsupported message type", "type", mt.String())
		return
	}
	reply, err := dhcpv4.NewReplyFromRequest(d.Pkt, mods...)
	if err != nil {
		m.Log.Error(err, "error creating reply")
		return
	}
	cm := &ipv4.ControlMessage{IfIndex: d.Md.IfIndex}
	if _, err := conn.WriteTo(reply.ToBytes(), cm, d.Peer); err != nil {
		m.Log.Error(err, "failed to send reply")
		return
	}
	m.Log.Info("sent reply")
}

func (m *mock) setOpts() []dhcpv4.Modifier {
	mods := []dhcpv4.Modifier{
		dhcpv4.WithGeneric(dhcpv4.OptionServerIdentifier, m.ServerIP),
		dhcpv4.WithServerIP(m.ServerIP),
		dhcpv4.WithLeaseTime(m.LeaseTime),
		dhcpv4.WithYourIP(m.YourIP),
		dhcpv4.WithDNS(m.NameServers...),
		dhcpv4.WithNetmask(m.SubnetMask),
		dhcpv4.WithRouter(m.Router),
	}

	return mods
}

func dhcp(ctx context.Context) (*dhcpv4.DHCPv4, error) {
	rifs, err := nettest.RoutedInterface("ip", net.FlagUp|net.FlagBroadcast)
	if err != nil {
		return nil, err
	}
	c, err := nclient4.New(rifs.Name,
		nclient4.WithServerAddr(&net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7676}),
		nclient4.WithUnicast(&net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7677}),
	)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return c.DiscoverOffer(ctx)
}

func TestServe(t *testing.T) {
	tests := map[string]struct {
		h    Handler
		addr netip.AddrPort
	}{
		"success": {addr: netip.MustParseAddrPort("127.0.0.1:7676"), h: &mock{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := NewServer("lo", net.UDPAddrFromAddrPort(tt.addr), tt.h)
			if err != nil {
				t.Fatal(err)
			}
			ctx, done := context.WithCancel(context.Background())
			defer done()

			go s.Serve(ctx)

			// make client calls
			d, err := dhcp(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(d)

			done()
		})
	}
}
