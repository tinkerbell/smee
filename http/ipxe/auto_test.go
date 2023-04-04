package ipxe

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/dhcp/data"
	"go.opentelemetry.io/otel/trace"
)

func buildCustomScript(s string) string {
	return fmt.Sprintf("#!ipxe\n\necho Loading custom Tinkerbell iPXE script...\n%s\n", s)
}

func TestCustomScript(t *testing.T) {
	tests := map[string]struct {
		input *data.Netboot
		want  string
		err   bool
	}{
		"custom script":      {input: &data.Netboot{IPXEScript: "echo custom script\nautoboot"}, want: "echo custom script\nautoboot"},
		"custom chain":       {input: &data.Netboot{IPXEScriptURL: &url.URL{Scheme: "http", Host: "127.0.0.1", Path: "/ipxe"}}, want: "chain --autofree http://127.0.0.1/ipxe"},
		"nil input":          {input: nil, err: true},
		"no script or chain": {input: &data.Netboot{}, err: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := &ScriptHandler{}
			got, err := h.customScript(tt.input)
			if tt.err && err == nil {
				t.Fatal("expected error")
			}
			if err != nil && !tt.err {
				t.Fatalf("did not expect error: %v", err)
			}

			if !tt.err {
				if diff := cmp.Diff(got, buildCustomScript(tt.want)); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestDefaultScript(t *testing.T) {
	hook := `#!ipxe

set syslog 127.0.0.1

echo Loading the Tinkerbell Hook iPXE script...
echo Debug TraceID: 00010203040506070000000000000000

set arch x86_64
set download-url http://127.0.0.1:8080/osie

kernel ${download-url}/vmlinuz-${arch} vlan_id=100 console=ttyS0 \
facility=onprem syslog_host=127.0.0.1 grpc_authority=127.0.0.1:42113 tinkerbell_tls=false worker_id=00:00:00:00:00:00 hw_addr=00:00:00:00:00:00 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`
	tests := map[string]struct {
		input  *data.DHCP
		input2 *data.Netboot
		want   string
	}{
		"no facility": {input: &data.DHCP{Arch: "x86_64", MACAddress: []byte{0o0, 0o0, 0o0, 0o0, 0o0, 0o0}, VLANID: "100"}, input2: &data.Netboot{Facility: "onprem"}, want: hook},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := &ScriptHandler{
				Logger:             logr.Discard(),
				OSIEURL:            "http://127.0.0.1:8080/osie",
				ExtraKernelParams:  []string{"console=ttyS0"},
				PublicSyslogFQDN:   "127.0.0.1",
				TinkServerTLS:      false,
				TinkServerGRPCAddr: "127.0.0.1:42113",
			}
			span := trace.NewSpanContext(trace.SpanContextConfig{TraceID: [16]byte{0, 1, 2, 3, 4, 5, 6, 7}, TraceFlags: trace.FlagsSampled})
			ctx := trace.ContextWithSpanContext(context.Background(), span)

			got, err := h.defaultScript(trace.SpanFromContext(ctx), tt.input, tt.input2)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

type mockBackend struct {
	netboot *data.Netboot
	dhcp    *data.DHCP
}

func (m *mockBackend) GetByIP(_ context.Context, _ net.IP) (*data.DHCP, *data.Netboot, error) {
	return m.dhcp, m.netboot, nil
}

func (m *mockBackend) GetByMac(_ context.Context, _ net.HardwareAddr) (*data.DHCP, *data.Netboot, error) {
	return m.dhcp, m.netboot, nil
}

func TestHandlerFunc(t *testing.T) {
	hook := `#!ipxe

set syslog 127.0.0.1

echo Loading the Tinkerbell Hook iPXE script...

set arch x86_64
set download-url http://127.0.0.1:8080/osie

kernel ${download-url}/vmlinuz-${arch} vlan_id=100 console=ttyS0 \
facility=onprem syslog_host=127.0.0.1 grpc_authority=127.0.0.1:42113 tinkerbell_tls=false worker_id=00:00:00:00:00:00 hw_addr=00:00:00:00:00:00 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`
	tests := map[string]struct {
		dhcp    *data.DHCP
		netboot *data.Netboot
		want    string
		err     bool
	}{
		"no netboot":     {dhcp: &data.DHCP{}, netboot: &data.Netboot{AllowNetboot: false}, err: true},
		"default script": {dhcp: &data.DHCP{Arch: "x86_64", MACAddress: []byte{0o0, 0o0, 0o0, 0o0, 0o0, 0o0}, VLANID: "100"}, netboot: &data.Netboot{AllowNetboot: true, Facility: "onprem"}, want: hook},
	}
	metrics.Init()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mockBackend{netboot: tt.netboot, dhcp: tt.dhcp}
			h := &ScriptHandler{
				Logger:             logr.Discard(),
				Backend:            m,
				OSIEURL:            "http://127.0.0.1:8080/osie",
				ExtraKernelParams:  []string{"console=ttyS0"},
				PublicSyslogFQDN:   "127.0.0.1",
				TinkServerTLS:      false,
				TinkServerGRPCAddr: "127.0.0.1:42113",
			}

			fn := h.HandlerFunc()
			ts := httptest.NewServer(fn)
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(got), tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
