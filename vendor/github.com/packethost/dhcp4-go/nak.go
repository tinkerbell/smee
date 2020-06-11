package dhcp4

import "encoding/binary"

// Nak is a server to client packet indicating client's notion of network
// address is incorrect (e.g., client has moved to new subnet) or client's
// lease as expired.
type Nak struct {
	Packet

	msg *Packet
}

func CreateNak(msg *Packet) Nak {
	rep := Nak{
		Packet: NewReply(msg),
		msg:    msg,
	}

	rep.SetMessageType(MessageTypeNak)
	return rep
}

// From RFC2131, table 3:
//   Option                    DHCPNAK
//   ------                    -------
//   Requested IP address      MUST NOT
//   IP address lease time     MUST NOT
//   Use 'file'/'sname' fields MUST NOT
//   DHCP message type         DHCPNAK
//   Parameter request list    MUST NOT
//   Message                   SHOULD
//   Client identifier         MAY
//   Vendor class identifier   MAY
//   Server identifier         MUST
//   Maximum message size      MUST NOT
//   All others                MUST NOT

var dhcpNakAllowedOptions = []Option{
	OptionDHCPMsgType,
	OptionDHCPMessage,
	OptionClientID,
	OptionClassID,
	OptionDHCPServerID,
}

var dhcpNakValidation = []Validation{
	ValidateMust(OptionDHCPServerID),
	ValidateAllowedOptions(dhcpNakAllowedOptions),
}

func (d *Nak) Validate() error {
	return Validate(d.Packet, dhcpNakValidation)
}

func (d *Nak) ToBytes() ([]byte, error) {
	opts := packetToBytesOptions{
		skipFile:  true,
		skipSName: true,
	}

	// Copy MaxMsgSize if set in the request
	if v, ok := d.Message().GetOption(OptionDHCPMaxMsgSize); ok {
		opts.maxLen = binary.BigEndian.Uint16(v)
	}

	return PacketToBytes(d.Packet, &opts)
}

func (d *Nak) Message() *Packet {
	return d.msg
}

func (d *Nak) Reply() *Packet {
	return &d.Packet
}
