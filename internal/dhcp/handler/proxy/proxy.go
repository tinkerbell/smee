/*
Package proxy implements a DHCP handler that provides proxyDHCP functionality.

"[A] Proxy DHCP server behaves much like a DHCP server by listening for ordinary
DHCP client traffic and responding to certain client requests. However, unlike the
DHCP server, the PXE Proxy DHCP server does not administer network addresses, and
it only responds to clients that identify themselves as PXE clients. The responses
given by the PXE Proxy DHCP server contain the mechanism by which the client locates
the boot servers or the network addresses and descriptions of the supported,
compatible boot servers."

Reference: https://www.ibm.com/docs/en/aix/7.1?topic=protocol-preboot-execution-environment-proxy-dhcp-daemon
*/
package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/tinkerbell/smee/internal/dhcp"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
	oteldhcp "github.com/tinkerbell/smee/internal/dhcp/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/ipv4"
)

const tracerName = "github.com/tinkerbell/smee/internal/dhcp/handler/proxy"

// Handler holds the configuration details for the running the DHCP server.
type Handler struct {
	// Backend is the backend to use for getting DHCP data.
	Backend handler.BackendReader

	// IPAddr is the IP address to use in DHCP responses.
	// Option 54 and the sname DHCP header.
	// This could be a load balancer IP address or an ingress IP address or a local IP address.
	IPAddr netip.Addr

	// Log is used to log messages.
	// `logr.Discard()` can be used if no logging is desired.
	Log logr.Logger

	// Netboot configuration
	Netboot Netboot

	// OTELEnabled is used to determine if netboot options include otel naming.
	// When true, the netboot filename will be appended with otel information.
	// For example, the filename will be "snp.efi-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01".
	// <original filename>-00-<trace id>-<span id>-<trace flags>
	OTELEnabled bool
}

// Netboot holds the netboot configuration details used in running a DHCP server.
type Netboot struct {
	// iPXE binary server IP:Port serving via TFTP.
	IPXEBinServerTFTP netip.AddrPort

	// IPXEBinServerHTTP is the URL to the IPXE binary server serving via HTTP(s).
	IPXEBinServerHTTP *url.URL

	// IPXEScriptURL is the URL to the IPXE script to use.
	IPXEScriptURL func(*dhcpv4.DHCPv4) *url.URL

	// Enabled is whether to enable sending netboot DHCP options.
	Enabled bool

	// UserClass (for network booting) allows a custom DHCP option 77 to be used to break out of an iPXE loop.
	UserClass dhcp.UserClass
}

// Redirection name comes from section 2.5 of http://www.pix.net/software/pxeboot/archive/pxespec.pdf
func (h *Handler) Handle(ctx context.Context, conn *ipv4.PacketConn, dp data.Packet) {
	// validations
	if dp.Pkt == nil {
		h.Log.Error(errors.New("incoming packet is nil"), "not able to respond when the incoming packet is nil")
		return
	}
	upeer, ok := dp.Peer.(*net.UDPAddr)
	if !ok {
		h.Log.Error(errors.New("peer is not a UDP connection"), "not able to respond when the peer is not a UDP connection")
		return
	}
	if upeer == nil {
		h.Log.Error(errors.New("peer is nil"), "not able to respond when the peer is nil")
		return
	}
	if conn == nil {
		h.Log.Error(errors.New("connection is nil"), "not able to respond when the connection is nil")
		return
	}

	var ifName string
	if dp.Md != nil {
		ifName = dp.Md.IfName
	}
	log := h.Log.WithValues("mac", dp.Pkt.ClientHWAddr.String(), "xid", dp.Pkt.TransactionID.String(), "interface", ifName)
	tracer := otel.Tracer(tracerName)
	var span trace.Span
	ctx, span = tracer.Start(
		ctx,
		fmt.Sprintf("DHCP Packet Received: %v", dp.Pkt.MessageType().String()),
		trace.WithAttributes(h.encodeToAttributes(dp.Pkt, "request")...),
		trace.WithAttributes(attribute.String("DHCP.peer", dp.Peer.String())),
		trace.WithAttributes(attribute.String("DHCP.server.ifname", ifName)),
	)

	defer span.End()

	// We ignore the error here because:
	// 1. it's only non-nil if the generation of a transaction id (XID) fails.
	// 2. We always use the clients transaction id (XID) in responses. See dhcpv4.WithReply().
	reply, _ := dhcpv4.NewReplyFromRequest(dp.Pkt)

	if dp.Pkt.OpCode != dhcpv4.OpcodeBootRequest { // TODO(jacobweinstock): dont understand this, found it in an example here: https://github.com/insomniacslk/dhcp/blob/c51060810aaab9c8a0bd1b0fcbf72bc0b91e6427/dhcpv4/server4/server_test.go#L31
		log.V(1).Info("Ignoring packet", "OpCode", dp.Pkt.OpCode)
		span.SetStatus(codes.Ok, "Ignoring packet: OpCode not BootRequest")

		return
	}

	if err := setMessageType(reply, dp.Pkt.MessageType()); err != nil {
		log.V(1).Info("Ignoring packet", "error", err.Error())
		span.SetStatus(codes.Ok, err.Error())

		return
	}

	// Set option 97
	reply.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, dp.Pkt.GetOneOption(dhcpv4.OptionClientMachineIdentifier)))

	i := dhcp.NewInfo(dp.Pkt)

	if !h.Netboot.Enabled {
		log.V(1).Info("Ignoring packet: netboot is not enabled")
		span.SetStatus(codes.Ok, "Ignoring packet: netboot is not enabled")

		return
	}
	if err := i.IsNetbootClient; err != nil {
		log.V(1).Info("Ignoring packet: not from a PXE enabled client", "error", err.Error())
		span.SetStatus(codes.Ok, fmt.Sprintf("Ignoring packet: not from a PXE enabled client: %s", err.Error()))

		return
	}
	if i.IPXEBinary == "" {
		log.V(1).Info("Ignoring packet: no iPXE binary was able to be determined")
		span.SetStatus(codes.Ok, "Ignoring packet: no iPXE binary was able to be determined")

		return
	}

	// Set option 43
	opts := dhcpv4.Options{6: []byte{8}} // PXE Boot Server Discovery Control - bypass, just boot from dhcp header: bootfile. No need to set opt for tftp server address.
	reply.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionVendorSpecificInformation, i.AddRPIOpt43(opts)))

	// Set option 60
	// The PXE spec says the server should identify itself as a PXEClient or HTTPClient
	reply.UpdateOption(dhcpv4.OptClassIdentifier(i.ClientTypeFrom().String()))

	// Set option 54, without this the pxe client will try to broadcast a request message to port 4011 for the ipxe binary. only found to be needed for PXEClient but not prohibitive for HTTPClient.
	// probably will want this to be the public IP of the proxyDHCP server
	ns := i.NextServer(h.Netboot.IPXEBinServerHTTP, h.Netboot.IPXEBinServerTFTP)
	reply.UpdateOption(dhcpv4.OptServerIdentifier(ns))
	// add the siaddr (IP address of next server) dhcp packet header to a given packet pkt.
	// see https://datatracker.ietf.org/doc/html/rfc2131#section-2
	// without this the pxe client will try to broadcast a request message to port 4011 for the ipxe script. The value doesnt seem to matter.
	reply.ServerIPAddr = ns

	// set sname header
	// see https://datatracker.ietf.org/doc/html/rfc2131#section-2
	reply.ServerHostName = ns.String()
	// setSNAME(reply, dp.Pkt.GetOneOption(dhcpv4.OptionClassIdentifier), h.Netboot.IPXEBinServerTFTP.Addr().AsSlice(), net.ParseIP(h.Netboot.IPXEBinServerHTTP.Hostname()))

	// set bootfile header
	reply.BootFileName = i.Bootfile("", h.Netboot.IPXEScriptURL(dp.Pkt), h.Netboot.IPXEBinServerHTTP, h.Netboot.IPXEBinServerTFTP)

	// check the backend, if PXE is NOT allowed, set the boot file name to "/<mac address>/not-allowed"
	_, n, err := h.Backend.GetByMac(ctx, dp.Pkt.ClientHWAddr)
	if err != nil || (n != nil && !n.AllowNetboot) {
		l := log.V(1)
		if err != nil {
			l = l.WithValues("error", err.Error())
		}
		if n != nil {
			l = l.WithValues("netbootAllowed", n.AllowNetboot)
		}
		l.Info("Ignoring packet")
		span.SetStatus(codes.Ok, "netboot not allowed")
		return
	}
	log.Info(
		"received DHCP packet",
		"type", dp.Pkt.MessageType().String(),
		"clientType", i.ClientTypeFrom().String(),
		"userClass", i.UserClassFrom().String(),
	)

	dst := replyDestination(dp.Peer, dp.Pkt.GatewayIPAddr)
	cm := &ipv4.ControlMessage{}
	if dp.Md != nil {
		cm.IfIndex = dp.Md.IfIndex
	}
	log = log.WithValues(
		"destination", dst.String(),
		"bootFileName", reply.BootFileName,
		"nextServer", reply.ServerIPAddr.String(),
		"messageType", reply.MessageType().String(),
		"serverHostname", reply.ServerHostName,
	)
	// send the DHCP packet
	if _, err := conn.WriteTo(reply.ToBytes(), cm, dst); err != nil {
		log.Error(err, "failed to send ProxyDHCP response")
		span.SetStatus(codes.Error, err.Error())

		return
	}
	log.Info("Sent ProxyDHCP response")
	span.SetAttributes(h.encodeToAttributes(reply, "reply")...)
	span.SetStatus(codes.Ok, "sent DHCP response")
}

// encodeToAttributes takes a DHCP packet and returns opentelemetry key/value attributes.
func (h *Handler) encodeToAttributes(d *dhcpv4.DHCPv4, namespace string) []attribute.KeyValue {
	a := &oteldhcp.Encoder{Log: h.Log}

	return a.Encode(d, namespace, oteldhcp.AllEncoders()...)
}

func setMessageType(reply *dhcpv4.DHCPv4, reqMsg dhcpv4.MessageType) error {
	switch mt := reqMsg; mt {
	case dhcpv4.MessageTypeDiscover:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	case dhcpv4.MessageTypeRequest:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	default:
		return IgnorePacketError{PacketType: mt, Details: "proxyDHCP only responds to Discover or Request message types"}
	}
	return nil
}

// IgnorePacketError is for when a DHCP packet should be ignored.
type IgnorePacketError struct {
	PacketType dhcpv4.MessageType
	Details    string
}

// Error returns the string representation of ErrIgnorePacket.
func (e IgnorePacketError) Error() string {
	return fmt.Sprintf("Ignoring packet: message type %s: details %s", e.PacketType, e.Details)
}

// replyDestination determines the destination address for the DHCP reply.
// If the giaddr is set, then the reply should be sent to the giaddr.
// Otherwise, the reply should be sent to the direct peer.
//
// From page 22 of https://www.ietf.org/rfc/rfc2131.txt:
// "If the 'giaddr' field in a DHCP message from a client is non-zero,
// the server sends any return messages to the 'DHCP server' port on
// the BOOTP relay agent whose address appears in 'giaddr'.".
func replyDestination(directPeer net.Addr, giaddr net.IP) net.Addr {
	if !giaddr.IsUnspecified() && giaddr != nil {
		return &net.UDPAddr{IP: giaddr, Port: dhcpv4.ServerPort}
	}

	return directPeer
}
