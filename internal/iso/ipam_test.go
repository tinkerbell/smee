package iso

import (
	"net"
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/smee/internal/dhcp/data"
)

func TestParseIPAM(t *testing.T) {
	tests := map[string]struct {
		input *data.DHCP
		want  string
	}{
		"empty": {},
		"only MAC": {
			input: &data.DHCP{MACAddress: net.HardwareAddr{0xde, 0xed, 0xbe, 0xef, 0xfe, 0xed}},
			want:  "ipam=de-ed-be-ef-fe-ed::::::::",
		},
		"everything": {
			input: &data.DHCP{
				MACAddress:     net.HardwareAddr{0xde, 0xed, 0xbe, 0xef, 0xfe, 0xed},
				IPAddress:      netip.AddrFrom4([4]byte{127, 0, 0, 1}),
				SubnetMask:     net.IPv4Mask(255, 255, 255, 0),
				DefaultGateway: netip.AddrFrom4([4]byte{127, 0, 0, 2}),
				NameServers:    []net.IP{{1, 1, 1, 1}, {4, 4, 4, 4}},
				Hostname:       "myhost",
				NTPServers:     []net.IP{{129, 6, 15, 28}, {129, 6, 15, 29}},
				DomainSearch:   []string{"example.com", "example.org"},
				VLANID:         "400",
			},
			want: "ipam=de-ed-be-ef-fe-ed:400:127.0.0.1:255.255.255.0:127.0.0.2:myhost:1.1.1.1,4.4.4.4:example.com,example.org:129.6.15.28,129.6.15.29",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseIPAM(tt.input)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("diff: %v", diff)
			}
		})
	}
}
