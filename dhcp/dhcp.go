package dhcp

import (
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
)

type Reply interface {
	Packet() *dhcp4.Packet
	Send() error
}

func NewReply(w dhcp4.ReplyWriter, req *dhcp4.Packet) Reply {
	switch req.GetMessageType() {
	case dhcp4.MessageTypeDiscover:
		return NewOffer(w, req)
	case dhcp4.MessageTypeRequest:
		return NewAck(w, req)
	}

	return nil
}

type Ack struct {
	dhcp4.Ack
	w dhcp4.ReplyWriter
}

func NewAck(w dhcp4.ReplyWriter, req *dhcp4.Packet) *Ack {
	ack := dhcp4.CreateAck(req)
	includeOption82(req, ack)

	return &Ack{ack, w}
}

func (r *Ack) Packet() *dhcp4.Packet {
	return &r.Ack.Packet
}

func (r *Ack) Send() error {
	return errors.Wrap(r.w.WriteReply(&r.Ack), "failed to write ACK")
}

type Offer struct {
	dhcp4.Offer
	w dhcp4.ReplyWriter
}

func NewOffer(w dhcp4.ReplyWriter, req *dhcp4.Packet) *Offer {
	offer := dhcp4.CreateOffer(req)
	includeOption82(req, offer)

	return &Offer{offer, w}
}

func (r *Offer) Packet() *dhcp4.Packet {
	return &r.Offer.Packet
}

func (r *Offer) Send() error {
	return errors.Wrap(r.w.WriteReply(&r.Offer), "failed to write OFFER")
}

func includeOption82(req *dhcp4.Packet, res dhcp4.OptionSetter) {
	// check if option 82 exists
	if opt82, ok := req.GetOption(dhcp4.OptionRelayAgentInformation); ok {
		// copy it to response
		res.SetOption(dhcp4.OptionRelayAgentInformation, opt82)
	}
}
