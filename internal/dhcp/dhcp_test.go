package dhcp

import (
	"encoding/hex"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
)

const (
	examplePXEClient  = "PXEClient:Arch:00007:UNDI:003001"
	exampleHTTPClient = "HTTPClient:Arch:00016:UNDI:003001"
)

func TestNewInfo(t *testing.T) {
	tests := map[string]struct {
		pkt  *dhcpv4.DHCPv4
		want Info
	}{
		"valid http client": {
			pkt: &dhcpv4.DHCPv4{
				ClientIPAddr: []byte{0x00, 0x00, 0x00, 0x00},
				ClientHWAddr: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				Options: dhcpv4.OptionsFromList(
					dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
					dhcpv4.OptClientArch(iana.EFI_X86_64_HTTP),
					dhcpv4.OptUserClass(Tinkerbell.String()),
					dhcpv4.OptClassIdentifier(exampleHTTPClient),
					dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}),
					dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{0x00, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x05, 0x06, 0x07}),
				),
			},
			want: Info{
				Arch:            iana.EFI_X86_64_HTTP,
				Mac:             net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				UserClass:       Tinkerbell,
				ClientType:      HTTPClient,
				IsNetbootClient: nil,
				IPXEBinary:      "ipxe.efi",
			},
		},
		"arch not found": {
			pkt: &dhcpv4.DHCPv4{
				ClientIPAddr: []byte{0x00, 0x00, 0x00, 0x00},
				ClientHWAddr: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				Options: dhcpv4.OptionsFromList(
					dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
					dhcpv4.OptClientArch(iana.Arch(255)),
					dhcpv4.OptClassIdentifier(examplePXEClient),
					dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}),
					dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{0x00, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x05, 0x06, 0x07}),
				),
			},
			want: Info{
				Arch:       iana.Arch(255),
				Mac:        net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				ClientType: PXEClient,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := NewInfo(tt.pkt)
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(Info{}, "Pkt")); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestArch(t *testing.T) {
	tests := map[string]struct {
		pkt  *dhcpv4.DHCPv4
		want iana.Arch
	}{
		"found": {
			pkt:  &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(dhcpv4.OptClientArch(iana.INTEL_X86PC))},
			want: iana.INTEL_X86PC,
		},
		"raspberry pi": {
			pkt:  &dhcpv4.DHCPv4{ClientHWAddr: net.HardwareAddr{0xb8, 0x27, 0xeb, 0x00, 0x00, 0x00}},
			want: iana.Arch(41),
		},
		"unknown": {
			pkt:  &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(dhcpv4.OptClientArch(iana.Arch(255)))},
			want: iana.Arch(255),
		},
		"unknown: opt 93 len 0": {
			pkt:  &dhcpv4.DHCPv4{},
			want: iana.Arch(255),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := Arch(tt.pkt)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestIsNetbootClient(t *testing.T) {
	tests := map[string]struct {
		input *dhcpv4.DHCPv4
		want  error
	}{
		"fail invalid message type": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(dhcpv4.OptMessageType(dhcpv4.MessageTypeInform))}, want: errors.New("")},
		"fail no opt60":             {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover))}, want: errors.New("")},
		"fail bad opt60": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("BadClient"),
		)}, want: errors.New("")},
		"fail no opt93": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("HTTPClient:Arch:xxxxx:UNDI:yyyzzz"),
		)}, want: errors.New("")},
		"fail no opt94": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("HTTPClient:Arch:xxxxx:UNDI:yyyzzz"),
			dhcpv4.OptClientArch(iana.EFI_ARM64_HTTP),
		)}, want: errors.New("")},
		"fail invalid opt97[0] != 0": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("HTTPClient:Arch:xxxxx:UNDI:yyyzzz"),
			dhcpv4.OptClientArch(iana.EFI_ARM64_HTTP),
			dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}),
			dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x00, 0x02, 0x03, 0x04, 0x05, 0x06, 0x00, 0x02, 0x03, 0x04, 0x05}),
		)}, want: errors.New("")},
		"fail invalid len(opt97)": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("HTTPClient:Arch:xxxxx:UNDI:yyyzzz"),
			dhcpv4.OptClientArch(iana.EFI_ARM64_HTTP),
			dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}),
			dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{0x01, 0x02}),
		)}, want: errors.New("")},
		"success len(opt97) == 0": {input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeDiscover),
			dhcpv4.OptClassIdentifier("HTTPClient:Arch:xxxxx:UNDI:yyyzzz"),
			dhcpv4.OptClientArch(iana.EFI_ARM64_HTTP),
			dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}),
			dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{}),
		)}, want: nil},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := IsNetbootClient(tt.input); (err == nil) != (tt.want == nil) {
				t.Errorf("isNetbootClient() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBootfile(t *testing.T) {
	type args struct {
		customUC          UserClass
		ipxeTFTPBinServer netip.AddrPort
		ipxeScript        *url.URL
		ipxeHTTPBinServer *url.URL
	}
	tests := map[string]struct {
		info Info
		args args
		want string
	}{
		"ipxe script": {
			info: Info{
				UserClass: Tinkerbell,
			},
			args: args{
				ipxeScript: &url.URL{Path: "/ipxe-script"},
			},
			want: "/ipxe-script",
		},
		"http client": {
			info: Info{
				ClientType: HTTPClient,
				Mac:        net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				IPXEBinary: "ipxe.efi",
			},
			args: args{
				ipxeHTTPBinServer: &url.URL{Scheme: "http", Host: "1.2.3.4:8080"},
			},
			want: "http://1.2.3.4:8080/01:02:03:04:05:06/ipxe.efi",
		},
		"firmware ipxe": {
			info: Info{
				UserClass:  IPXE,
				Mac:        net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				IPXEBinary: "undionly.kpxe",
			},
			args: args{
				ipxeTFTPBinServer: netip.MustParseAddrPort("1.2.3.4:69"),
			},
			want: "tftp://1.2.3.4:69/01:02:03:04:05:06/undionly.kpxe",
		},
		"no user class": {
			info: Info{
				Mac:        net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
				IPXEBinary: "undionly.kpxe",
			},
			want: "undionly.kpxe",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.info.Bootfile(tt.args.customUC, tt.args.ipxeScript, tt.args.ipxeHTTPBinServer, tt.args.ipxeTFTPBinServer)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNextServer(t *testing.T) {
	type args struct {
		ipxeTFTPBinServer netip.AddrPort
		ipxeHTTPBinServer *url.URL
	}
	tests := map[string]struct {
		info Info
		args args
		want net.IP
	}{
		"http client": {
			info: Info{
				ClientType: HTTPClient,
			},
			args: args{
				ipxeHTTPBinServer: &url.URL{Scheme: "http", Host: "1.2.3.4:8989"},
			},
			want: net.ParseIP("1.2.3.4"),
		},
		"firmware ipxe": {
			info: Info{
				UserClass: IPXE,
			},
			args: args{
				ipxeTFTPBinServer: netip.MustParseAddrPort("1.2.3.4:69"),
			},
			want: net.ParseIP("1.2.3.4"),
		},
		"no user class": {
			info: Info{},
			args: args{
				ipxeTFTPBinServer: netip.MustParseAddrPort("1.2.3.4:69"),
			},
			want: net.ParseIP("1.2.3.4"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.info.NextServer(tt.args.ipxeHTTPBinServer, tt.args.ipxeTFTPBinServer)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestOpt43(t *testing.T) {
	rpi9, _ := hex.DecodeString("00001152617370626572727920506920426f6f74")
	rpi10, _ := hex.DecodeString("00505845")

	tests := map[string]struct {
		info Info
		opts dhcpv4.Options
		want []byte
	}{
		"not a raspberry pi": {
			info: Info{
				Mac: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			},
			opts: dhcpv4.Options{},
			want: dhcpv4.Options{}.ToBytes(),
		},
		"raspberry pi": {
			info: Info{
				Mac: net.HardwareAddr{0xb8, 0x27, 0xeb, 0x00, 0x00, 0x00},
			},
			opts: dhcpv4.Options{},
			want: dhcpv4.Options{
				9:  rpi9,
				10: rpi10,
			}.ToBytes(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.info.AddRPIOpt43(tt.opts)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestUserClassString(t *testing.T) {
	u := UserClass("test")
	if diff := cmp.Diff("test", u.String()); diff != "" {
		t.Fatal(diff)
	}
}

func TestIsRaspberryPI(t *testing.T) {
	tests := map[string]struct {
		mac  net.HardwareAddr
		want bool
	}{
		"not a raspberry pi": {
			mac:  net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			want: false,
		},
		"raspberry pi": {
			mac:  net.HardwareAddr{0xb8, 0x27, 0xeb, 0x00, 0x00, 0x00},
			want: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := isRaspberryPI(tt.mac)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
