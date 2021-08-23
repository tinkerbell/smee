package main

import (
	"os"
	"testing"

	"github.com/packethost/dhcp4-go"
	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/metrics"
)

func TestGetCircuitID(t *testing.T) {

	for _, test := range []struct {
		name        string
		option      dhcp4.Option
		optionvalue []byte
		expected    string
		err         string // logged error description
	}{
		{
			name:        "With option82 circuitid",
			option:      dhcp4.OptionRelayAgentInformation,
			optionvalue: []byte("\x01\x19esr1.d11.lab1:ge-1/0/47.0\x02\x0Bge-1/0/47.0"),
			expected:    "esr1.d11.lab1:ge-1/0/47",
			err:         "",
		},
		{
			name:        "No option82 information",
			option:      dhcp4.OptionEnd, // option not important here, just needs to not have OptionRelayAgentInformation
			optionvalue: []byte{},
			expected:    "",
			err:         "option82 information not available for this mac",
		},
		{
			name:        "Malformed option82",
			option:      dhcp4.OptionRelayAgentInformation,
			optionvalue: []byte("\x01\x19esr1.d11.la"),
			expected:    "",
			err:         "option82 option1 out of bounds (check eightytwo[1])",
		},
	} {
		t.Log(test.name)
		var packet = new(dhcp4.Packet)

		packet.OptionMap = make(dhcp4.OptionMap, 255)
		packet.SetOption(test.option, test.optionvalue)
		c, err := getCircuitID(packet)
		if err != nil {
			if err.Error() != test.err {
				t.Fatalf("unexpected error, want: %s, got: %s", test.err, err)
			}
		}
		if c != "" {
			if c != test.expected {
				t.Fatalf("expected value not returned for option82, want: %s, got: %s", test.expected, c)
			}
		}

	}
}

func TestMain(m *testing.M) {
	l, err := log.Init("github.com/tinkerbell/boots")
	if err != nil {
		panic(nil)
	}
	defer l.Close()
	mainlog = l.Package("main")
	metrics.Init(l)
	os.Exit(m.Run())
}
