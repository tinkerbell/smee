package dhcp4

import (
	"errors"
	"net"
)

var (
	ErrShortPacket   = errors.New("dhcp4: short packet")
	ErrInvalidPacket = errors.New("dhcp4: invalid packet")
)

type OpCode byte

// Message op codes defined in RFC2132.
const (
	BootRequest = OpCode(1)
	BootReply   = OpCode(2)
)

type PacketGetter interface {
	GetHType() uint8
	GetHLen() uint8
	GetXID() []byte
	GetFlags() []byte
	GetCHAddr() net.HardwareAddr

	GetCIAddr() net.IP
	GetYIAddr() net.IP
	GetSIAddr() net.IP
	GetGIAddr() net.IP
}

type PacketSetter interface {
	SetCIAddr(ip net.IP)
	SetYIAddr(ip net.IP)
	SetSIAddr(ip net.IP)
	SetGIAddr(ip net.IP)
}

type RawPacket []byte

func (p RawPacket) Op() []byte     { return p[0:1] }
func (p RawPacket) HType() []byte  { return p[1:2] }
func (p RawPacket) HLen() []byte   { return p[2:3] }
func (p RawPacket) Hops() []byte   { return p[3:4] }
func (p RawPacket) XID() []byte    { return p[4:8] }
func (p RawPacket) Secs() []byte   { return p[8:10] }
func (p RawPacket) Flags() []byte  { return p[10:12] }
func (p RawPacket) CIAddr() []byte { return p[12:16] }
func (p RawPacket) YIAddr() []byte { return p[16:20] }
func (p RawPacket) SIAddr() []byte { return p[20:24] }
func (p RawPacket) GIAddr() []byte { return p[24:28] }
func (p RawPacket) CHAddr() []byte { return p[28:44] }

// SName returns the `sname` portion of the packet.
// This field can be used as extra space to extend the DHCP options, if
// necessary. To enable this, the "Option Overload" option needs to be set in
// the regular options. Also see RFC2132, section 9.3.
func (p RawPacket) SName() []byte {
	return p[44:108]
}

// File returns the `file` portion of the packet.
// This field can be used as extra space to extend the DHCP options, if
// necessary. To enable this, the "Option Overload" option needs to be set in
// the regular options. Also see RFC2132, section 9.3.
func (p RawPacket) File() []byte {
	return p[108:236]
}

// Cookie returns the fixed-value prefix to the `options` portion of the packet.
// According to the RFC, this should equal the 4-octet { 99, 130, 83, 99 }.
func (p RawPacket) Cookie() []byte {
	return p[236:240]
}

// Options returns the variable-sized `options` portion of the packet.
func (p RawPacket) Options() []byte {
	return p[240:]
}

// GetHType gets the hardware address type.
func (p RawPacket) GetHType() uint8 {
	return uint8(p.HType()[0])
}

// GetHLen gets the hardware address length.
func (p RawPacket) GetHLen() uint8 {
	return uint8(p.HLen()[0])
}

// GetXID gets the packet's transaction ID.
func (p RawPacket) GetXID() []byte {
	var out [4]byte
	copy(out[:], p.XID())
	return out[:]
}

// GetFlags gets the packet's flags.
func (p RawPacket) GetFlags() []byte {
	var out [2]byte
	copy(out[:], p.Flags())
	return out[:]
}

// GetCHAddr gets the client's hardware address.
func (p RawPacket) GetCHAddr() net.HardwareAddr {
	var out [16]byte

	hlen := p.GetHLen()
	if hlen > 16 {
		hlen = 16
	}

	copy(out[:], p.CHAddr()[0:hlen])
	return net.HardwareAddr(out[0:hlen])
}

// GetCIAddr gets the current IP address of the client.
func (p RawPacket) GetCIAddr() net.IP {
	return net.IP(p.CIAddr())
}

// SetCIAddr sets the current IP address of the client.
//
// From RFC2131 section 3.5:
// The client fills in the 'ciaddr' field only when correctly configured with
// an IP address in BOUND, RENEWING or REBINDING state.
func (p RawPacket) SetCIAddr(ip net.IP) {
	copy(p.CIAddr(), ip)
}

// GetYIAddr gets the IP address offered or assigned to the client.
func (p RawPacket) GetYIAddr() net.IP {
	return net.IP(p.YIAddr())
}

// SetYIAddr sets the IP address offered or assigned to the client.
//
// From RFC2131 section 3.1:
// Each server may respond with a DHCPOFFER message that includes an available
// network address in the 'yiaddr' field.
func (p RawPacket) SetYIAddr(ip net.IP) {
	copy(p.YIAddr(), ip)
}

// GetSIAddr gets the IP address of the next server to use in bootstrap.
func (p RawPacket) GetSIAddr() net.IP {
	return net.IP(p.SIAddr())
}

// SetSIAddr sets the IP address of the next server to use in bootstrap.
//
// From RFC2131 section 2: DHCP clarifies the interpretation of the 'siaddr'
// field as the address of the server to use in the next step of the client's
// bootstrap process. Returned in DHCPOFFER, DHCPACK by server.
func (p RawPacket) SetSIAddr(ip net.IP) {
	copy(p.SIAddr(), ip)
}

// GetGIAddr gets the IP address of the relay agent.
func (p RawPacket) GetGIAddr() net.IP {
	return net.IP(p.GIAddr())
}

// SetGIAddr sets the IP address of the relay agent.
//
// From RFC2131 section 2: Relay agent IP address, used in booting via a relay
// agent.
func (p RawPacket) SetGIAddr(ip net.IP) {
	copy(p.GIAddr(), ip)
}

func (p RawPacket) ParseOptions() (OptionMap, error) {
	var err error

	// Facilitate up to 255 option tags
	opts := make(OptionMap, 255)

	// Parse initial set of options
	if err = opts.Deserialize(p.Options(), nil); err != nil {
		return nil, err
	}

	// Parse options from `file` field if necessary
	if x := opts[OptionOverload]; len(x) > 0 && x[0]&0x1 != 0 {
		if err = opts.Deserialize(p.File(), nil); err != nil {
			return nil, err
		}
	}

	// Parse options from `sname` field if necessary
	if x := opts[OptionOverload]; len(x) > 0 && x[0]&0x2 != 0 {
		if err = opts.Deserialize(p.SName(), nil); err != nil {
			return nil, err
		}
	}

	return opts, nil
}

type Packet struct {
	RawPacket
	OptionMap
}

// NewPacket creates and returns a new packet with the specified OpCode.
func NewPacket(o OpCode) Packet {
	p := Packet{
		RawPacket: make([]byte, 240),
		OptionMap: make(OptionMap),
	}

	copy(p.Op(), []byte{byte(o)})
	copy(p.Cookie(), []byte{99, 130, 83, 99})

	return p
}

// NewReply creates and returns a new reply packet given a request.
func NewReply(msg PacketGetter) Packet {
	rep := NewPacket(BootReply)

	// Hardware type and address length
	rep.HType()[0] = 1 // Ethernet
	rep.HLen()[0] = 6  // MAC-48 is 6 octets

	// Copy transaction identifier
	copy(rep.XID(), msg.GetXID()[:])

	// Copy fields from request (per RFC2131, section 4.3, table 3)
	copy(rep.Flags(), msg.GetFlags())
	copy(rep.CHAddr(), msg.GetCHAddr())
	copy(rep.GIAddr(), msg.GetGIAddr())

	// The remainder of the fields are set depending on the outcome of the
	// handler. Once the packet has been filled in, it should be validated before
	// sending it out on the wire.
	return rep
}

// PacketFromBytes deserializes the wire-level representation of a DHCP packet
// contained in the []byte b into a Packet struct. The function returns an
// error if the packet is malformed. The contents of []byte b is copied into
// the resulting structure and can be reused after this function has returned.
func PacketFromBytes(b []byte) (Packet, error) {
	var err error

	if len(b) < 240 {
		return Packet{}, ErrShortPacket
	}

	p := Packet{
		RawPacket: make(RawPacket, len(b)),
	}

	copy(p.RawPacket, b)

	p.OptionMap, err = p.ParseOptions()
	if err != nil {
		return Packet{}, err
	}

	return p, nil
}

type packetToBytesOptions struct {
	maxLen    uint16
	skipFile  bool
	skipSName bool
}

// PacketToBytes serializes the DHCP packet pointed to by p into its wire-level
// representation. The function may return an error if it cannot successfully
// serialize the packet. Otherwise, it returns a newly created byte slice.
func PacketToBytes(p Packet, opts *packetToBytesOptions) ([]byte, error) {
	if len(p.RawPacket) < 240 {
		return nil, ErrInvalidPacket
	}

	// Maximum byte length of serialized packet (default is Ethernet MTU).
	var maxLen uint16 = 1500

	// The mininum "Maximum DHCP Message Size" is 576 (RFC2132, 9.10).
	if opts != nil && opts.maxLen > 576 {
		maxLen = opts.maxLen
	}

	// Buffers we can stash options in
	var b [3][]byte

	// Variable length options field (starting at byte 240)
	b[0] = make([]byte, 0, maxLen-240)

	// Fixed length "file" field (from byte 108 to byte 236)
	if opts == nil || !opts.skipFile {
		b[1] = make([]byte, 0, 236-108)
	}

	// Fixed length "sname" field (from byte 44 to byte 108)
	if opts == nil || !opts.skipSName {
		b[2] = make([]byte, 0, 108-44)
	}

	// Write options to one of the buffers.
	// Iterate over options in numeric order.
	for _, k := range p.GetSortedOptions() {
		v := p.OptionMap[k]
		l := 2 + len(v)

		// TODO(PN): Deal with DHCP options of length > 255
		// https://www.pivotaltracker.com/story/show/68123382
		if len(v) > 255 {
			continue
		}

		for i := range b {
			cb := cap(b[i])
			lb := len(b[i])
			f := cb - lb

			// The first buffer needs to have at least 3 bytes extra for OptionOverload
			if i == 0 {
				f -= 3
			}

			// Every buffer needs to have at least 1 byte extra for OptionEnd
			f--

			// Check that this buffer has room for this option
			if f < l {
				continue
			}

			// Write option to buffer
			b[i] = b[i][:lb+l]
			b[i][lb+0] = byte(k)
			b[i][lb+1] = byte(len(v))
			copy(b[i][lb+2:], v)
			break
		}
	}

	// Add OptionEnd to the buffers that need one
	for i := range b {
		lb := len(b[i])
		if i == 0 || lb > 0 {
			b[i] = b[i][:lb+1]
			b[i][lb] = byte(OptionEnd)
		}
	}

	// Capacity: base packet, optional OptionOverload option, and options field
	oc := 240 + 3 + len(b[0])
	ol := 0
	o := make([]byte, ol, oc)

	// Copy base packet
	copy(o[0:240], p.RawPacket[0:240])
	ol = 240

	// Copy options overloaded into the SName and File sections
	if len(b[1]) > 0 || len(b[2]) > 0 {
		overload := 0x0

		// File section
		if len(b[1]) > 0 {
			overload |= 0x1
			copy(o[108:236], b[1])
		}

		// SName section
		if len(b[2]) > 0 {
			overload |= 0x2
			copy(o[44:108], b[2])
		}

		// Add OptionOverload
		o = o[:ol+3]
		o[ol+0] = byte(OptionOverload)
		o[ol+1] = byte(1)
		o[ol+2] = byte(overload)
		ol += 3
	}

	// Add options
	o = o[:ol+len(b[0])]
	copy(o[ol:ol+len(b[0])], b[0])

	return o, nil
}
