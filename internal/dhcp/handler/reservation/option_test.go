package reservation

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
	"github.com/insomniacslk/dhcp/rfc1035label"
	"github.com/tinkerbell/smee/internal/dhcp"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	dhcpotel "github.com/tinkerbell/smee/internal/otel"
	oteldhcp "github.com/tinkerbell/smee/internal/dhcp/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	examplePXEClient  = "PXEClient:Arch:00007:UNDI:003001"
	exampleHTTPClient = "HTTPClient:Arch:00016:UNDI:003001"
)

func TestSetDHCPOpts(t *testing.T) {
	type args struct {
		in0 context.Context
		m   *dhcpv4.DHCPv4
		d   *data.DHCP
	}
	tests := map[string]struct {
		server Handler
		args   args
		want   *dhcpv4.DHCPv4
	}{
		"success": {
			server: Handler{Log: logr.Discard(), SyslogAddr: netip.MustParseAddr("192.168.7.7")},
			args: args{
				in0: context.Background(),
				m:   &dhcpv4.DHCPv4{Options: dhcpv4.OptionsFromList(dhcpv4.OptParameterRequestList(dhcpv4.OptionSubnetMask))},
				d: &data.DHCP{
					MACAddress:     net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
					IPAddress:      netip.MustParseAddr("192.168.4.4"),
					SubnetMask:     []byte{255, 255, 255, 0},
					DefaultGateway: netip.MustParseAddr("192.168.4.1"),
					NameServers: []net.IP{
						{8, 8, 8, 8},
						{8, 8, 4, 4},
					},
					Hostname:         "test-server",
					DomainName:       "mynet.local",
					BroadcastAddress: netip.MustParseAddr("192.168.4.255"),
					NTPServers: []net.IP{
						{132, 163, 96, 2},
						{132, 163, 96, 3},
					},
					LeaseTime: 84600,
					DomainSearch: []string{
						"mynet.local",
					},
				},
			},
			want: &dhcpv4.DHCPv4{
				OpCode:        dhcpv4.OpcodeBootRequest,
				HWType:        iana.HWTypeEthernet,
				ClientHWAddr:  net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				ClientIPAddr:  []byte{0, 0, 0, 0},
				YourIPAddr:    []byte{192, 168, 4, 4},
				ServerIPAddr:  []byte{0, 0, 0, 0},
				GatewayIPAddr: []byte{0, 0, 0, 0},
				Options: dhcpv4.OptionsFromList(
					dhcpv4.OptGeneric(dhcpv4.OptionLogServer, []byte{192, 168, 7, 7}),
					dhcpv4.OptSubnetMask(net.IPMask{255, 255, 255, 0}),
					dhcpv4.OptBroadcastAddress(net.IP{192, 168, 4, 255}),
					dhcpv4.OptIPAddressLeaseTime(time.Duration(84600)*time.Second),
					dhcpv4.OptDomainName("mynet.local"),
					dhcpv4.OptHostName("test-server"),
					dhcpv4.OptRouter(net.IP{192, 168, 4, 1}),
					dhcpv4.OptDNS([]net.IP{
						{8, 8, 8, 8},
						{8, 8, 4, 4},
					}...),
					dhcpv4.OptNTPServers([]net.IP{
						{132, 163, 96, 2},
						{132, 163, 96, 3},
					}...),
					dhcpv4.OptDomainSearch(&rfc1035label.Labels{
						Labels: []string{"mynet.local"},
					}),
				),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := &Handler{
				Log: tt.server.Log,
				Netboot: Netboot{
					IPXEBinServerTFTP: tt.server.Netboot.IPXEBinServerTFTP,
					IPXEBinServerHTTP: tt.server.Netboot.IPXEBinServerHTTP,
					IPXEScriptURL:     tt.server.Netboot.IPXEScriptURL,
					Enabled:           tt.server.Netboot.Enabled,
					UserClass:         tt.server.Netboot.UserClass,
				},
				IPAddr:     tt.server.IPAddr,
				Backend:    tt.server.Backend,
				SyslogAddr: tt.server.SyslogAddr,
			}
			mods := s.setDHCPOpts(tt.args.in0, tt.args.m, tt.args.d)
			finalPkt, err := dhcpv4.New(mods...)
			if err != nil {
				t.Fatalf("setDHCPOpts() error = %v, wantErr nil", err)
			}
			if diff := cmp.Diff(tt.want, finalPkt, cmpopts.IgnoreFields(dhcpv4.DHCPv4{}, "TransactionID")); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestBootfileAndNextServer(t *testing.T) {
	type args struct {
		pkt     *dhcpv4.DHCPv4
		uClass  dhcp.UserClass
		tftp    netip.AddrPort
		ipxe    *url.URL
		iscript *url.URL
	}
	tests := map[string]struct {
		server       *Handler
		args         args
		otelEnabled  bool
		wantBootFile string
		wantNextSrv  net.IP
	}{
		"success bootfile only": {
			server: &Handler{Log: logr.Discard()},
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptUserClass(dhcp.Tinkerbell.String()),
					),
				},
				iscript: &url.URL{Scheme: "http", Host: "localhost:8080", Path: "/auto.ipxe"},
			},
			wantBootFile: "http://localhost:8080/auto.ipxe",
			wantNextSrv:  nil,
		},
		"success httpClient": {
			server: &Handler{Log: logr.Discard()},
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptClientArch(iana.EFI_ARM64_HTTP),
						dhcpv4.OptClassIdentifier(exampleHTTPClient),
					),
				},
				ipxe: &url.URL{Scheme: "http", Host: "127.0.0.1:8181"},
			},
			wantBootFile: "http://127.0.0.1:8181/01:02:03:04:05:06/snp.efi",
			wantNextSrv:  net.IPv4(127, 0, 0, 1),
		},
		"success userclass iPXE": {
			server: &Handler{Log: logr.Discard()},
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x07},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptClientArch(iana.INTEL_X86PC),
						dhcpv4.OptUserClass(dhcp.IPXE.String()),
					),
				},
				tftp: netip.MustParseAddrPort("192.168.6.5:69"),
			},
			wantBootFile: "tftp://192.168.6.5:69/01:02:03:04:05:07/undionly.kpxe",
			wantNextSrv:  net.ParseIP("192.168.6.5"),
		},
		"success userclass iPXE with otel": {
			server:      &Handler{Log: logr.Discard(), OTELEnabled: true},
			otelEnabled: true,
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x07},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptClientArch(iana.INTEL_X86PC),
						dhcpv4.OptUserClass(dhcp.IPXE.String()),
					),
				},
				tftp: netip.MustParseAddrPort("192.168.6.5:69"),
				ipxe: &url.URL{Scheme: "tftp", Host: "192.168.6.5:69"},
			},
			wantBootFile: "tftp://192.168.6.5:69/01:02:03:04:05:07/undionly.kpxe-00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01",
			wantNextSrv:  net.ParseIP("192.168.6.5"),
		},
		"success default": {
			server: &Handler{Log: logr.Discard()},
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x07},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptClientArch(iana.INTEL_X86PC),
					),
				},
				tftp: netip.MustParseAddrPort("192.168.6.5:69"),
				ipxe: &url.URL{Scheme: "tftp", Host: "192.168.6.5:69"},
			},
			wantBootFile: "undionly.kpxe",
			wantNextSrv:  net.ParseIP("192.168.6.5"),
		},
		"success otel enabled, no traceparent": {
			server: &Handler{Log: logr.Discard(), OTELEnabled: true},
			args: args{
				pkt: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x07},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptClientArch(iana.INTEL_X86PC),
					),
				},
				tftp: netip.MustParseAddrPort("192.168.6.5:69"),
				ipxe: &url.URL{Scheme: "tftp", Host: "192.168.6.5:69"},
			},
			wantBootFile: "undionly.kpxe",
			wantNextSrv:  net.ParseIP("192.168.6.5"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if tt.otelEnabled {
				// set global propagator to tracecontext (the default is no-op).
				prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
				otel.SetTextMapPropagator(prop)
				ctx = dhcpotel.ContextWithTraceparentString(ctx, "00-23b1e307bb35484f535a1f772c06910e-d887dc3912240434-01")
			}
			bootfile, nextServer := tt.server.bootfileAndNextServer(ctx, tt.args.pkt, tt.args.uClass, tt.args.tftp, tt.args.ipxe, tt.args.iscript)
			if diff := cmp.Diff(bootfile, tt.wantBootFile); diff != "" {
				t.Fatal("bootfile", diff)
			}
			if diff := cmp.Diff(nextServer, tt.wantNextSrv); diff != "" {
				t.Fatal("nextServer", diff)
			}
		})
	}
}

func TestSetNetworkBootOpts(t *testing.T) {
	type args struct {
		in0 context.Context
		m   *dhcpv4.DHCPv4
		n   *data.Netboot
	}
	tests := map[string]struct {
		server *Handler
		args   args
		want   *dhcpv4.DHCPv4
	}{
		"netboot not allowed": {
			server: &Handler{Log: logr.Discard()},
			args: args{
				in0: context.Background(),
				m:   &dhcpv4.DHCPv4{},
				n:   &data.Netboot{AllowNetboot: false},
			},
			want: &dhcpv4.DHCPv4{ServerIPAddr: net.IPv4(0, 0, 0, 0), BootFileName: "/netboot-not-allowed"},
		},
		"netboot allowed": {
			server: &Handler{Log: logr.Discard(), Netboot: Netboot{IPXEScriptURL: func(*dhcpv4.DHCPv4) *url.URL {
				return &url.URL{Scheme: "http", Host: "localhost:8181", Path: "/01:02:03:04:05:06/auto.ipxe"}
			}}},
			args: args{
				in0: context.Background(),
				m: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptUserClass(dhcp.Tinkerbell.String()),
						dhcpv4.OptClassIdentifier("HTTPClient:xxxxx"),
						dhcpv4.OptClientArch(iana.EFI_X86_64_HTTP),
					),
				},
				n: &data.Netboot{AllowNetboot: true, IPXEScriptURL: &url.URL{Scheme: "http", Host: "localhost:8181", Path: "/01:02:03:04:05:06/auto.ipxe"}},
			},
			want: &dhcpv4.DHCPv4{BootFileName: "http://localhost:8181/01:02:03:04:05:06/auto.ipxe", Options: dhcpv4.OptionsFromList(
				dhcpv4.OptGeneric(dhcpv4.OptionVendorSpecificInformation, dhcpv4.Options{
					6:  []byte{8},
					69: oteldhcp.TraceparentFromContext(context.Background()),
				}.ToBytes()),
				dhcpv4.OptClassIdentifier("HTTPClient"),
			)},
		},
		"netboot not allowed, arch unknown": {
			server: &Handler{Log: logr.Discard(), Netboot: Netboot{IPXEScriptURL: func(*dhcpv4.DHCPv4) *url.URL {
				return &url.URL{Scheme: "http", Host: "localhost:8181", Path: "/01:02:03:04:05:06/auto.ipxe"}
			}}},
			args: args{
				in0: context.Background(),
				m: &dhcpv4.DHCPv4{
					ClientHWAddr: net.HardwareAddr{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					Options: dhcpv4.OptionsFromList(
						dhcpv4.OptUserClass(dhcp.Tinkerbell.String()),
						dhcpv4.OptClientArch(iana.UBOOT_ARM64),
					),
				},
				n: &data.Netboot{AllowNetboot: true},
			},
			want: &dhcpv4.DHCPv4{ServerIPAddr: net.IPv4(0, 0, 0, 0), BootFileName: "/netboot-not-allowed"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := &Handler{
				Log: tt.server.Log,
				Netboot: Netboot{
					IPXEBinServerTFTP: tt.server.Netboot.IPXEBinServerTFTP,
					IPXEBinServerHTTP: tt.server.Netboot.IPXEBinServerHTTP,
					IPXEScriptURL:     tt.server.Netboot.IPXEScriptURL,
					Enabled:           tt.server.Netboot.Enabled,
					UserClass:         tt.server.Netboot.UserClass,
				},
				IPAddr:  tt.server.IPAddr,
				Backend: tt.server.Backend,
			}
			gotFunc := s.setNetworkBootOpts(tt.args.in0, tt.args.m, tt.args.n)
			got := new(dhcpv4.DHCPv4)
			gotFunc(got)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
