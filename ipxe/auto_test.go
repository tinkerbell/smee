package ipxe

import (
	"context"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/boots/client"
	"go.opentelemetry.io/otel/trace"
)

func TestCustomScriptFound(t *testing.T) {
	tests := map[string]struct {
		ipxeURL       string
		ipeScript     string
		nilDiscoverer bool
		want          bool
	}{
		"found script":            {want: true, ipeScript: "#!ipxe\nautoboot"},
		"found url":               {want: true, ipxeURL: "https://boot.netboot.xyz"},
		"not found":               {want: false},
		"no Discoverer interface": {want: false, nilDiscoverer: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hw := &client.DiscovererMock{
				GetMACFunc: func(ip net.IP) net.HardwareAddr {
					return net.HardwareAddr{}
				},
				HardwareFunc: func() client.Hardware {
					return &client.HardwareMock{
						IPXEScriptFunc: func(mac net.HardwareAddr) string {
							return tt.ipeScript
						},
						IPXEURLFunc: func(mac net.HardwareAddr) string {
							return tt.ipxeURL
						},
					}
				},
			}
			var got bool
			if tt.nilDiscoverer {
				got = customScriptFound(nil, "127.0.0.1")
			} else {
				got = customScriptFound(hw, "127.0.0.1")
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}

func TestCustomScript(t *testing.T) {
	tests := map[string]struct {
		ipxeURL   string
		ipeScript string
		want      string
		shouldErr bool
	}{
		"got script":         {want: "#!ipxe\n\necho Loading custom Tinkerbell iPXE script...\n#!ipxe\nautoboot\n", ipeScript: "#!ipxe\nautoboot"},
		"got url":            {want: "#!ipxe\n\necho Loading custom Tinkerbell iPXE script...\nchain --autofree https://boot.netboot.xyz\n", ipxeURL: "https://boot.netboot.xyz"},
		"invalid URL prefix": {want: "", ipxeURL: "invalid", shouldErr: true},
		"invalid URL":        {want: "", ipxeURL: "http://invalid.:123.com", shouldErr: true},
		"no script or url":   {want: "", shouldErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hw := &client.DiscovererMock{
				GetMACFunc: func(ip net.IP) net.HardwareAddr {
					return net.HardwareAddr{}
				},
				HardwareFunc: func() client.Hardware {
					return &client.HardwareMock{
						IPXEScriptFunc: func(mac net.HardwareAddr) string {
							return tt.ipeScript
						},
						IPXEURLFunc: func(mac net.HardwareAddr) string {
							return tt.ipxeURL
						},
					}
				},
			}
			h := &Handler{}
			got, err := h.customScript(hw, "127.0.0.1")
			if err != nil && !tt.shouldErr {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}

func TestDefaultScript(t *testing.T) {
	one := `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...

set arch x86_64
set download-url http://127.1.1.1

kernel ${download-url}/vmlinuz-${arch} vlan_id=1234 \
facility=onprem syslog_host= grpc_authority= tinkerbell_tls=false worker_id=00:01:02:03:04:05 hw_addr=00:01:02:03:04:05 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`
	tests := map[string]struct {
		want string
	}{
		"success with defaults": {want: one},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hw := &client.DiscovererMock{
				GetMACFunc: func(ip net.IP) net.HardwareAddr {
					return []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
				},
				HardwareFunc: func() client.Hardware {
					return &client.HardwareMock{
						IPXEScriptFunc: func(mac net.HardwareAddr) string {
							return ""
						},
						IPXEURLFunc: func(mac net.HardwareAddr) string {
							return ""
						},
						HardwareArchFunc: func(mac net.HardwareAddr) string {
							return "x86_64"
						},
						HardwareFacilityCodeFunc: func() string {
							return "onprem"
						},
						GetVLANIDFunc: func(mac net.HardwareAddr) string {
							return "1234"
						},
					}
				},
				InstanceFunc: func() *client.Instance {
					return &client.Instance{}
				},
			}
			h := &Handler{
				OSIEURL: "http://127.1.1.1",
			}
			sp := trace.SpanFromContext(context.Background())
			got, err := h.defaultScript(sp, hw, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Log(got)
				t.Fatalf(diff)
			}
		})
	}
}
