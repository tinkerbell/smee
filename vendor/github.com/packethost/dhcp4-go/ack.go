package dhcp4

import "encoding/binary"

// Ack is a server to client packet with configuration parameters,
// including committed network address.
type Ack struct {
	Packet

	msg *Packet
}

func CreateAck(msg *Packet) Ack {
	rep := Ack{
		Packet: NewReply(msg),
		msg:    msg,
	}

	rep.SetMessageType(MessageTypeAck)
	return rep
}

// From RFC2131, table 3:
//   Option                    DHCPACK
//   ------                    -------
//   Requested IP address      MUST NOT
//   IP address lease time     MUST (DHCPREQUEST)
//                             MUST NOT (DHCPINFORM)
//   Use 'file'/'sname' fields MAY
//   DHCP message type         DHCPACK
//   Parameter request list    MUST NOT
//   Message                   SHOULD
//   Client identifier         MUST NOT
//   Vendor class identifier   MAY
//   Server identifier         MUST
//   Maximum message size      MUST NOT
//   All others                MAY

var dhcpAckOnRequestValidation = []Validation{
	ValidateMust(OptionAddressTime),
}

var dhcpAckOnInformValidation = []Validation{
	ValidateMustNot(OptionAddressTime),
}

var dhcpAckValidation = []Validation{
	ValidateMustNot(OptionAddressRequest),
	ValidateMustNot(OptionParameterList),
	ValidateMustNot(OptionClientID),
	ValidateMust(OptionDHCPServerID),
	ValidateMustNot(OptionDHCPMaxMsgSize),
}

func (d *Ack) Validate() error {
	var err error

	// Validation is subtly different based on type of request
	switch d.msg.GetMessageType() {
	case MessageTypeRequest:
		err = Validate(d.Packet, dhcpAckOnRequestValidation)
	case MessageTypeInform:
		err = Validate(d.Packet, dhcpAckOnInformValidation)
	}

	if err != nil {
		return err
	}

	return Validate(d.Packet, dhcpAckValidation)
}

func (d *Ack) ToBytes() ([]byte, error) {
	opts := packetToBytesOptions{}

	// Copy MaxMsgSize if set in the request
	if v, ok := d.Message().GetOption(OptionDHCPMaxMsgSize); ok {
		opts.maxLen = binary.BigEndian.Uint16(v)
	}

	return PacketToBytes(d.Packet, &opts)
}

func (d *Ack) Message() *Packet {
	return d.msg
}

func (d *Ack) Reply() *Packet {
	return &d.Packet
}
