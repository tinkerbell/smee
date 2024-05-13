package script

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace"
)

func TestCustomScript(t *testing.T) {
	tests := map[string]struct {
		ipxeURL    string
		ipxeScript string
		want       string
		shouldErr  bool
	}{
		"got script":         {want: "#!ipxe\n\necho Loading custom Tinkerbell iPXE script...\n#!ipxe\nautoboot\n", ipxeScript: "#!ipxe\nautoboot"},
		"got url":            {want: "#!ipxe\n\necho Loading custom Tinkerbell iPXE script...\nchain --autofree https://boot.netboot.xyz\n", ipxeURL: "https://boot.netboot.xyz"},
		"invalid URL prefix": {want: "", ipxeURL: "invalid", shouldErr: true},
		"invalid URL":        {want: "", ipxeURL: "http://invalid.:123.com", shouldErr: true},
		"no script or url":   {want: "", shouldErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := &Handler{}
			u, err := url.Parse(tt.ipxeURL)
			if err != nil && !tt.shouldErr {
				t.Fatal(err)
			}

			d := data{MACAddress: net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, IPXEScript: tt.ipxeScript, IPXEScriptURL: u}
			got, err := h.customScript(d)
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
set retries:int32 10

set idx:int32 0
:retry_kernel
kernel ${download-url}/vmlinuz-${arch} vlan_id=1234 \
facility=onprem syslog_host= grpc_authority= tinkerbell_tls=false worker_id=00:01:02:03:04:05 hw_addr=00:01:02:03:04:05 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200 || iseq ${idx} ${retries} && goto kernel-error || inc idx && goto retry_kernel

set idx:int32 0
:retry_initrd
initrd ${download-url}/initramfs-${arch} || iseq ${idx} ${retries} && goto initrd-error || inc idx && goto retry_initrd

set idx:int32 0
:retry_boot
boot || iseq ${idx} ${retries} && goto boot-error || inc idx && goto retry_boot

:kernel-error
echo Failed to load kernel
exit

:initrd-error
echo Failed to load initrd
exit

:boot-error
echo Failed to boot
exit
`
	tests := map[string]struct {
		want string
	}{
		"success with defaults": {want: one},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := &Handler{
				OSIEURL: "http://127.1.1.1",
				IPXEScriptRetries: 10,
			}
			d := data{MACAddress: net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, VLANID: "1234", Facility: "onprem", Arch: "x86_64"}
			sp := trace.SpanFromContext(context.Background())
			got, err := h.defaultScript(sp, d)
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
