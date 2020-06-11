package dhcp4

import "encoding/binary"

// Offer is a server to client packet in response to DHCPDISCOVER with
// offer of configuration parameters.
type Offer struct {
	Packet

	msg *Packet
}

func CreateOffer(msg *Packet) Offer {
	rep := Offer{
		Packet: NewReply(msg),
		msg:    msg,
	}

	rep.SetMessageType(MessageTypeOffer)
	return rep
}

// From RFC2131, table 3:
//   Option                    DHCPOFFER
//   ------                    ---------
//   Requested IP address      MUST NOT
//   IP address lease time     MUST
//   Use 'file'/'sname' fields MAY
//   DHCP message type         DHCPOFFER
//   Parameter request list    MUST NOT
//   Message                   SHOULD
//   Client identifier         MUST NOT
//   Vendor class identifier   MAY
//   Server identifier         MUST
//   Maximum message size      MUST NOT
//   All others                MAY

var dhcpOfferValidation = []Validation{
	ValidateMustNot(OptionAddressRequest),
	ValidateMust(OptionAddressTime),
	ValidateMustNot(OptionParameterList),
	ValidateMustNot(OptionClientID),
	ValidateMust(OptionDHCPServerID),
	ValidateMustNot(OptionDHCPMaxMsgSize),
}

func (d *Offer) Validate() error {
	return Validate(d.Packet, dhcpOfferValidation)
}

func (d *Offer) ToBytes() ([]byte, error) {
	opts := packetToBytesOptions{}

	// Copy MaxMsgSize if set in the request
	if v, ok := d.Message().GetOption(OptionDHCPMaxMsgSize); ok {
		opts.maxLen = binary.BigEndian.Uint16(v)
	}

	return PacketToBytes(d.Packet, &opts)
}

func (d *Offer) Message() *Packet {
	return d.msg
}

func (d *Offer) Reply() *Packet {
	return &d.Packet
}
