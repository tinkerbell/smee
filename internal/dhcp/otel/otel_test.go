package otel

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/insomniacslk/dhcp/rfc1035label"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestEncode(t *testing.T) {
	tests := map[string]struct {
		allEncoders bool
		pkt         *dhcpv4.DHCPv4
		want        []attribute.KeyValue
	}{
		"no encoders": {pkt: &dhcpv4.DHCPv4{}, want: nil},
		"all encoders": {allEncoders: true, pkt: &dhcpv4.DHCPv4{BootFileName: "ipxe.efi", Flags: 0}, want: []attribute.KeyValue{
			{Key: attribute.Key("DHCP.test.Header.flags"), Value: attribute.StringValue("Unicast")},
			{Key: attribute.Key("DHCP.test.Header.transactionID"), Value: attribute.StringValue("0x00000000")},
			{Key: attribute.Key("DHCP.test.Header.file"), Value: attribute.StringValue("ipxe.efi")},
		}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := &Encoder{}
			got := e.Encode(tt.pkt, "test")
			if tt.allEncoders {
				got = e.Encode(tt.pkt, "test", AllEncoders()...)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Logf("%+v", got)
				t.Fatal(diff)
			}
		})
	}
}

func TestEncodeError(t *testing.T) {
	tests := map[string]struct {
		input *notFoundError
		want  string
	}{
		"success":           {input: &notFoundError{optName: "opt1"}, want: "\"opt1\" not found in DHCP packet"},
		"success nil error": {input: &notFoundError{}, want: "\"\" not found in DHCP packet"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.input.Error()
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt1(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptSubnetMask(net.IPMask(net.IP{255, 255, 255, 0}.To4())),
			)},
			want: attribute.String("DHCP.testing.Opt1.SubnetMask", "255.255.255.0"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt1(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt1() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt3(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptRouter([]net.IP{{192, 168, 1, 1}}...),
			)},
			want: attribute.String("DHCP.testing.Opt3.DefaultGateway", "192.168.1.1"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt3(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt13() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt6(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptDNS([]net.IP{{1, 1, 1, 1}}...),
			)},
			want: attribute.String("DHCP.testing.Opt6.NameServers", "1.1.1.1"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt6(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt6() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt12(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptHostName("test-host"),
			)},
			want: attribute.String("DHCP.testing.Opt12.Hostname", "test-host"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt12(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt12() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt15(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptDomainName("example.com"),
			)},
			want: attribute.String("DHCP.testing.Opt15.DomainName", "example.com"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt15(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt15() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt28(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptBroadcastAddress(net.IP{192, 168, 1, 255}),
			)},
			want: attribute.String("DHCP.testing.Opt28.BroadcastAddress", "192.168.1.255"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt28(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt28() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt42(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptNTPServers([]net.IP{{132, 163, 96, 2}}...),
			)},
			want: attribute.String("DHCP.testing.Opt42.NTPServers", "132.163.96.2"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt42(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt42() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt51(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptIPAddressLeaseTime(time.Minute),
			)},
			want: attribute.String("DHCP.testing.Opt51.LeaseTime", "60"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt51(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt51() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt53(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer),
			)},
			want: attribute.String("DHCP.testing.Opt53.MessageType", "OFFER"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt53(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt53() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt54(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptServerIdentifier(net.IP{127, 0, 0, 1}),
			)},
			want: attribute.String("DHCP.testing.Opt54.ServerIdentifier", "127.0.0.1"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt54(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt54() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt60(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptClassIdentifier("foobar"),
			)},
			want: attribute.String("DHCP.testing.Opt60.ClassIdentifier", "foobar"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt60(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt60() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt93(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptClientArch(iana.INTEL_X86PC),
			)},
			want: attribute.StringSlice("DHCP.testing.Opt93.ClientIdentifier", []string{"Intel x86PC"}),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt93(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt93() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Log(tt.input.ClientArch())
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt94(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptGeneric(dhcpv4.OptionClientNetworkInterfaceIdentifier, []byte{0x01, 0x02, 0x01}),
			)},
			want: attribute.String("DHCP.testing.Opt94.ClientNetworkInterfaceIdentifier", "1.2.1"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt94(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt94() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Log(tt.input.ClientArch())
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt97(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptGeneric(dhcpv4.OptionClientMachineIdentifier, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}),
			)},
			want: attribute.String("DHCP.testing.Opt97.ClientMachineIdentifier", "0.1.2.3.4.5.6.7.8.9.10.11.12.13.14.15.16"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt97(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt97() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Log(tt.input.GetOneOption(dhcpv4.OptionClientMachineIdentifier))
				t.Fatal(diff)
			}
		})
	}
}

func TestSetOpt119(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(
				dhcpv4.OptDomainSearch(&rfc1035label.Labels{Labels: []string{"mydomain.com"}}),
			)},
			want: attribute.String("DHCP.testing.Opt119.DomainSearch", "mydomain.com"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeOpt119(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setOpt119() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderFlags(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{},
			want:  attribute.String("DHCP.testing.Header.flags", "Unicast"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeFlags(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setHeaderFlags() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderTransactionID(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{TransactionID: dhcpv4.TransactionID{0x00, 0x00, 0x00, 0x00}},
			want:  attribute.String("DHCP.testing.Header.transactionID", "0x00000000"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeTransactionID(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("EncodeTransactionID() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderYIADDR(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{YourIPAddr: []byte{192, 168, 2, 100}},
			want:  attribute.String("DHCP.testing.Header.yiaddr", "192.168.2.100"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeYIADDR(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setHeaderYIADDR() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderSIADDR(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{ServerIPAddr: []byte{127, 0, 0, 1}},
			want:  attribute.String("DHCP.testing.Header.siaddr", "127.0.0.1"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeSIADDR(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setHeaderSIADDR() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderCHADDR(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{ClientHWAddr: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}},
			want:  attribute.String("DHCP.testing.Header.chaddr", "01:02:03:04:05:06"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeCHADDR(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setHeaderCHADDR() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSetHeaderFILE(t *testing.T) {
	tests := map[string]struct {
		input   *dhcpv4.DHCPv4
		want    attribute.KeyValue
		wantErr error
	}{
		"success": {
			input: &dhcpv4.DHCPv4{BootFileName: "snp.efi"},
			want:  attribute.String("DHCP.testing.Header.file", "snp.efi"),
		},
		"error": {wantErr: &notFoundError{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeFILE(tt.input, "testing")
			if tt.wantErr != nil && !OptNotFound(err) {
				t.Fatalf("setHeaderFILE() error (type: %T) = %[1]v, wantErr (type: %T) %[2]v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(attribute.Value{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTraceparentFromContext(t *testing.T) {
	want := []byte{0, 1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 6, 7, 8, 0, 0, 0, 0, 1}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04},
		SpanID:     trace.SpanID{0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.TraceFlags(1),
	})
	rmSpan := trace.ContextWithRemoteSpanContext(context.Background(), sc)

	got := TraceparentFromContext(rmSpan)
	if !bytes.Equal(got, want) {
		t.Errorf("binaryTpFromContext() = %v, want %v", got, want)
	}
}
