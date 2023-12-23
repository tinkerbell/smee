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
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/tinkerbell/smee/internal/dhcp"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
	"golang.org/x/net/ipv4"
)

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

// netbootClient describes a device that is requesting a network boot.
type netbootClient struct {
	mac    net.HardwareAddr
	arch   iana.Arch
	uClass dhcp.UserClass
	cType  dhcp.ClientType
}

// Redirection name comes from section 2.5 of http://www.pix.net/software/pxeboot/archive/pxespec.pdf
func (h *Handler) Handle(ctx context.Context, conn *ipv4.PacketConn, data data.Packet) {
	log := h.Log.WithValues("hwaddr", data.Pkt.ClientHWAddr.String(), "listenAddr", conn.LocalAddr())
	reply, err := dhcpv4.New(dhcpv4.WithReply(data.Pkt),
		dhcpv4.WithGatewayIP(data.Pkt.GatewayIPAddr),
		dhcpv4.WithOptionCopied(data.Pkt, dhcpv4.OptionRelayAgentInformation),
	)
	if err != nil {
		log.Info("Generating a new transaction id failed, not a problem as we're passing one in, but if this message is showing up a lot then something could be up with github.com/insomniacslk/dhcp")
	}
	if data.Pkt.OpCode != dhcpv4.OpcodeBootRequest { // TODO(jacobweinstock): dont understand this, found it in an example here: https://github.com/insomniacslk/dhcp/blob/c51060810aaab9c8a0bd1b0fcbf72bc0b91e6427/dhcpv4/server4/server_test.go#L31
		log.V(1).Info("Ignoring packet", "OpCode", data.Pkt.OpCode)
		return
	}

	if err := dhcp.IsNetbootClient(data.Pkt); err != nil {
		log.V(1).Info("Ignoring packet: not from a PXE enabled client", "error", err.Error())
		return
	}

	if err := setMessageType(reply, data.Pkt.MessageType()); err != nil {
		log.V(1).Info("Ignoring packet", "error", err.Error())
		return
	}

	mach := process(data.Pkt)

	// Set option 43
	setOpt43(reply)

	// Set option 97, just copy from the incoming packet
	reply.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, data.Pkt.GetOneOption(dhcpv4.OptionClientMachineIdentifier)))

	// set broadcast header to true
	// reply.SetBroadcast()

	// Set option 60
	// The PXE spec says the server should identify itself as a PXEClient or HTTPCient
	if opt60 := data.Pkt.GetOneOption(dhcpv4.OptionClassIdentifier); strings.HasPrefix(string(opt60), string(dhcp.PXEClient)) {
		reply.UpdateOption(dhcpv4.OptClassIdentifier(string(dhcp.PXEClient)))
	} else {
		reply.UpdateOption(dhcpv4.OptClassIdentifier(string(dhcp.HTTPClient)))
	}

	// Set option 54
	opt54 := setOpt54(reply, data.Pkt.GetOneOption(dhcpv4.OptionClassIdentifier), h.Netboot.IPXEBinServerTFTP.Addr().AsSlice(), net.ParseIP(h.Netboot.IPXEBinServerHTTP.Hostname()))

	// add the siaddr (IP address of next server) dhcp packet header to a given packet pkt.
	// see https://datatracker.ietf.org/doc/html/rfc2131#section-2
	// without this the pxe client will try to broadcast a request message to port 4011
	reply.ServerIPAddr = opt54
	// probably will want this to be the public IP of the proxyDHCP server
	// reply.ServerIPAddr = h.Netboot.IPXEBinServerTFTP.Addr().AsSlice()

	// set sname header
	// see https://datatracker.ietf.org/doc/html/rfc2131#section-2
	setSNAME(reply, data.Pkt.GetOneOption(dhcpv4.OptionClassIdentifier), h.Netboot.IPXEBinServerTFTP.Addr().AsSlice(), net.ParseIP(h.Netboot.IPXEBinServerHTTP.Hostname()))

	// set bootfile header
	if err := setBootfile(reply, mach, h.Netboot.IPXEBinServerTFTP, h.Netboot.IPXEBinServerHTTP, h.Netboot.IPXEScriptURL(data.Pkt).String()); err != nil {
		log.Info("Ignoring packet", "error", err.Error())
		return
	}
	// check the backend, if PXE is NOT allowed, set the boot file name to "/<mac address>/not-allowed"
	_, n, err := h.Backend.GetByMac(context.Background(), data.Pkt.ClientHWAddr)
	if err != nil || (n != nil && !n.AllowNetboot) {
		log.V(1).Info("Ignoring packet", "error", err.Error(), "netbootAllowed", n)
		return
	}
	//if !h.Allower.Allow(, mach.mac) {
	//	rp.BootFileName = fmt.Sprintf("/%v/not-allowed", mach.mac)
	//}

	dst := replyDestination(data.Peer, data.Pkt.GatewayIPAddr)
	cm := &ipv4.ControlMessage{}
	if data.Md != nil {
		cm.IfIndex = data.Md.IfIndex
	}
	// send the DHCP packet
	if _, err := conn.WriteTo(reply.ToBytes(), cm, dst); err != nil {
		log.Error(err, "failed to send ProxyDHCP offer")
		return
	}
	//log.V(1).Info("DHCP packet received", "pkt", *data.Pkt)
	log.Info("Sent ProxyDHCP message", "arch", mach.arch, "userClass", mach.uClass, "receivedMsgType", data.Pkt.MessageType(), "replyMsgType", reply.MessageType(), "unicast", reply.IsUnicast(), "peer", dst, "bootfile", reply.BootFileName)
}

func setMessageType(reply *dhcpv4.DHCPv4, reqMsg dhcpv4.MessageType) error {
	switch mt := reqMsg; mt {
	case dhcpv4.MessageTypeDiscover:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	case dhcpv4.MessageTypeRequest:
		reply.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	default:
		return ErrIgnorePacket{PacketType: mt, Details: "proxyDHCP only responds to Discover or Request DHCP message types"}
	}
	return nil
}

// ErrIgnorePacket is for when a DHCP packet should be ignored.
type ErrIgnorePacket struct {
	PacketType dhcpv4.MessageType
	Details    string
}

// Error returns the string representation of ErrIgnorePacket.
func (e ErrIgnorePacket) Error() string {
	return fmt.Sprintf("Ignoring packet: message type %s: details %s", e.PacketType, e.Details)
}

// processMachine takes a DHCP packet and returns a populated machine struct.
func process(pkt *dhcpv4.DHCPv4) netbootClient {
	mach := netbootClient{}
	// get option 93 ; arch
	fwt := pkt.ClientArch()
	if len(fwt) == 0 {
		mach.arch = iana.Arch(255) // unassigned/unknown arch
	} else {
		// a netboot client may have multiple architectures in option 93
		// we will only handle the first one that is not unknown
		for _, elem := range fwt {
			if !strings.Contains(elem.String(), "unknown") {
				// Basic architecture identification, based purely on
				// the PXE architecture option.
				// https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml#processor-architecture
				mach.arch = elem
				break
			}
		}
	}

	// set option 77 from received packet
	mach.uClass = dhcp.UserClass(string(pkt.GetOneOption(dhcpv4.OptionUserClassInformation)))
	// set the client type based off of option 60
	opt60 := pkt.GetOneOption(dhcpv4.OptionClassIdentifier)
	if strings.HasPrefix(string(opt60), string(dhcp.PXEClient)) {
		mach.cType = dhcp.PXEClient
	} else if strings.HasPrefix(string(opt60), string(dhcp.HTTPClient)) {
		mach.cType = dhcp.HTTPClient
	}
	mach.mac = pkt.ClientHWAddr

	return mach
}

// setOpt43 is completely standard PXE: we tell the PXE client to
// bypass all the boot discovery rubbish that PXE supports,
// and just load a file from TFTP.
// TODO(jacobweinstock): add link to intel spec for this needing to be set.
func setOpt43(pkt *dhcpv4.DHCPv4) {
	// these are suboptions of option43. ref: https://datatracker.ietf.org/doc/html/rfc2132#section-8.4
	pxe := dhcpv4.Options{
		6: []byte{8}, // PXE Boot Server Discovery Control - bypass, just boot from filename.
	}
	// Raspberry PI's need options 9 and 10 of parent option 43.
	// The best way at the moment to figure out if a DHCP request is coming from a Raspberry PI is to
	// check the MAC address. We could reach out to some external server to tell us if the MAC address should
	// use these extra Raspberry PI options but that would require a dependency on some external service and all the trade-offs that
	// come with that. TODO: provide doc link for why these options are needed.
	// https://udger.com/resources/mac-address-vendor-detail?name=raspberry_pi_foundation
	h := strings.ToLower(pkt.ClientHWAddr.String())
	if strings.HasPrefix(h, strings.ToLower("B8:27:EB")) ||
		strings.HasPrefix(h, strings.ToLower("DC:A6:32")) ||
		strings.HasPrefix(h, strings.ToLower("E4:5F:01")) {
		// TODO document what these hex strings are and why they are needed.
		// https://www.raspberrypi.org/documentation/computers/raspberry-pi.html#PXE_OPTION43
		// tested with Raspberry Pi 4 using UEFI from here: https://github.com/pftf/RPi4/releases/tag/v1.31
		// all files were served via a tftp server and lived at the top level dir of the tftp server (i.e tftp://server/)
		// "\x00\x00\x11" is equal to NUL(Null), NUL(Null), DC1(Device Control 1)
		opt9, _ := hex.DecodeString("00001152617370626572727920506920426f6f74") // "\x00\x00\x11Raspberry Pi Boot"
		// "\x0a\x04\x00" is equal to LF(Line Feed), EOT(End of Transmission), NUL(Null)
		opt10, _ := hex.DecodeString("00505845") // "\x0a\x04\x00PXE"
		pxe[9] = opt9
		pxe[10] = opt10
	}

	pkt.UpdateOption(dhcpv4.OptGeneric(dhcpv4.OptionVendorSpecificInformation, pxe.ToBytes()))
}

// setOpt54 based on option 60. Also return the value for use other locations.
func setOpt54(reply *dhcpv4.DHCPv4, reqOpt60 []byte, tftp net.IP, http net.IP) net.IP {
	var opt54 net.IP
	if strings.HasPrefix(string(reqOpt60), string(dhcp.HTTPClient)) {
		opt54 = http
	} else {
		opt54 = tftp
	}
	reply.UpdateOption(dhcpv4.OptServerIdentifier(opt54))

	return opt54
}

// setSNAME sets the server hostname (sname) dhcp header.
func setSNAME(pkt *dhcpv4.DHCPv4, reqOpt60 []byte, tftp net.IP, http net.IP) {
	var sname string
	if strings.HasPrefix(string(reqOpt60), string(dhcp.HTTPClient)) {
		sname = http.String()
	} else {
		sname = tftp.String()
	}

	pkt.ServerHostName = sname
}

// setBootfile sets the setBootfile (file) dhcp header. see https://datatracker.ietf.org/doc/html/rfc2131#section-2 .
func setBootfile(reply *dhcpv4.DHCPv4, mach netbootClient, tftp netip.AddrPort, ipxe *url.URL, iscript string) error {
	// set bootfile header
	bin, found := dhcp.ArchToBootFile[mach.arch]
	if !found {
		return ErrArchNotFound{Arch: mach.arch}
	}
	var bootfile string
	// If a machine is in an ipxe boot loop, it is likely to be that we arent matching on IPXE or Tinkerbell.
	// if the "iPXE" user class is found it means we arent in our custom version of ipxe, but because of the option 43 we're setting we need to give a full tftp url from which to boot.
	switch { // order matters here.
	case mach.uClass == dhcp.Tinkerbell: // this case gets us out of an ipxe boot loop.
		bootfile = iscript
	case mach.cType == dhcp.HTTPClient: // Check the client type from option 60.
		bootfile = fmt.Sprintf("%s/%s/%s", ipxe, mach.mac.String(), bin)
	case mach.uClass == dhcp.IPXE:
		u := &url.URL{
			Scheme: "tftp",
			Host:   tftp.String(),
			Path:   fmt.Sprintf("%v/%v", mach.mac.String(), bin),
		}
		bootfile = u.String()
	default:
		bootfile = filepath.Join(mach.mac.String(), bin)
	}
	reply.BootFileName = bootfile

	return nil
}

// ErrArchNotFound is for when an PXE client request is an architecture that does not have a matching bootfile.
// See var ArchToBootFile for the look ups.
type ErrArchNotFound struct {
	Arch   iana.Arch
	Detail string
}

// Error returns the string representation of ErrArchNotFound.
func (e ErrArchNotFound) Error() string {
	return fmt.Sprintf("unable to find bootfile for arch %v: details %v", e.Arch, e.Detail)
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
