package main

import (
	"flag"

	"github.com/avast/retry-go"
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/job"
)

var listenAddr = conf.BOOTPBind

func init() {
	flag.StringVar(&listenAddr, "dhcp-addr", listenAddr, "IP and port to listen on for DHCP.")
}

// ServeDHCP is a useless comment
func ServeDHCP() {
	err := retry.Do(
		func() error {
			return errors.Wrap(dhcp4.ListenAndServe(listenAddr, dhcpHandler{}), "serving dhcp")
		},
	)
	if err != nil {
		mainlog.Fatal(errors.Wrap(err, "retry dhcp serve"))
	}
}

type dhcpHandler struct {
}

func (dhcpHandler) ServeDHCP(w dhcp4.ReplyWriter, req *dhcp4.Packet) {
	mac := req.GetCHAddr()
	if conf.ShouldIgnoreOUI(mac.String()) {
		mainlog.With("mac", mac).Info("mac is in ignore list")
		return
	}

	gi := req.GetGIAddr()
	if conf.ShouldIgnoreGI(gi.String()) {
		mainlog.With("giaddr", gi).Info("giaddr is in ignore list")
		return
	}

	circuitID, err := getCircuitID(req)
	if err != nil {
		mainlog.With("mac", mac, "err", err).Info("error parsing option82")
	} else {
		mainlog.With("mac", mac, "circuitID", circuitID).Info("parsed option82/circuitid")
	}

	j, err := job.CreateFromDHCP(mac, gi, circuitID)
	if err != nil {
		mainlog.With("type", req.GetMessageType(), "mac", mac, "err", err).Info("retrieved job is empty")
		return
	}
	go j.ServeDHCP(w, req)
}

func getCircuitID(req *dhcp4.Packet) (string, error) {
	var circuitID string
	// Pulling option82 information from the packet (this is the relaying router)
	// format: byte 1 is option number, byte 2 is length of the following array of bytes.
	eightytwo, ok := req.GetOption(dhcp4.OptionRelayAgentInformation)
	if ok {
		if int(eightytwo[1]) < len(eightytwo) {
			circuitID = string(eightytwo[2:eightytwo[1]])
		} else {
			return circuitID, errors.New("option82 option1 out of bounds (check eightytwo[1])")
		}
	}
	return circuitID, nil
}
