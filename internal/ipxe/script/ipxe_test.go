package script

import (
	"context"
	"net"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tinkerbell/smee/internal/metric"
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
				t.Fatal(diff)
			}
		})
	}
}

func TestDefaultScript(t *testing.T) {
	one := `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...

set arch x86_64
set download-url http://127.1.1.1
set kernel vmlinuz-${arch}
set initrd initramfs-${arch}
set retries:int32 10
set retry_delay:int32 3

set idx:int32 0
:retry_kernel
kernel ${download-url}/${kernel} vlan_id=1234 \
facility=onprem syslog_host= grpc_authority= tinkerbell_tls=false tinkerbell_insecure_tls=false worker_id=00:01:02:03:04:05 hw_addr=00:01:02:03:04:05 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200 && goto download_initrd || iseq ${idx} ${retries} && goto kernel-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_kernel

:download_initrd
set idx:int32 0
:retry_initrd
initrd ${download-url}/${initrd} && goto boot || iseq ${idx} ${retries} && goto initrd-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_initrd

:boot
set idx:int32 0
:retry_boot
boot || iseq ${idx} ${retries} && goto boot-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_boot

:kernel-error
echo Failed to load kernel
imgfree
exit

:initrd-error
echo Failed to load initrd
imgfree
exit

:boot-error
echo Failed to boot
imgfree
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
				OSIEURL:              "http://127.1.1.1",
				IPXEScriptRetries:    10,
				IPXEScriptRetryDelay: 3,
			}
			d := data{MACAddress: net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, VLANID: "1234", Facility: "onprem", Arch: "x86_64"}
			sp := trace.SpanFromContext(context.Background())
			got, err := h.defaultScript(sp, d)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Log(got)
				t.Fatal(diff)
			}
		})
	}
}

func TestStaticScript(t *testing.T) {
	want := `#!ipxe

echo Loading the static Tinkerbell iPXE script...

set arch ${buildarch}
# Tinkerbell only supports 64 bit archectures.
# The build architecture does not necessarily represent the architecture of the machine on which iPXE is running.
# https://ipxe.org/cfg/buildarch
iseq ${arch} i386 && set arch x86_64 ||
iseq ${arch} arm32 && set arch aarch64 ||
iseq ${arch} arm64 && set arch aarch64 ||
set download-url http://127.0.0.1
set retries:int32 0
set retry_delay:int32 0

set worker_id ${mac}
set grpc_authority 127.0.0.1:42113
set syslog_host 127.1.1.1
set tinkerbell_tls false

echo worker_id=${mac}
echo grpc_authority=127.0.0.1:42113
echo syslog_host=127.1.1.1
echo tinkerbell_tls=false

set idx:int32 0
:retry_kernel
kernel ${download-url}/vmlinuz-${arch} \
syslog_host=${syslog_host} grpc_authority=${grpc_authority} tinkerbell_tls=${tinkerbell_tls} worker_id=${worker_id} hw_addr=${mac} \
console=tty1 console=tty2 console=ttyAMA0,115200 console=ttyAMA1,115200 console=ttyS0,115200 console=ttyS1,115200 k=v k2=v2 \
intel_iommu=on iommu=pt k=v k2=v2 initrd=initramfs-${arch} && goto download_initrd || iseq ${idx} ${retries} && goto kernel-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_kernel

:download_initrd
set idx:int32 0
:retry_initrd
initrd ${download-url}/initramfs-${arch} && goto boot || iseq ${idx} ${retries} && goto initrd-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_initrd

:boot
set idx:int32 0
:retry_boot
boot || iseq ${idx} ${retries} && goto boot-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_boot

:kernel-error
echo Failed to load kernel
imgfree
exit

:initrd-error
echo Failed to load initrd
imgfree
exit

:boot-error
echo Failed to boot
imgfree
exit
`
	metric.Init()
	h := &Handler{
		OSIEURL:            "http://127.0.0.1",
		ExtraKernelParams:  []string{"k=v", "k2=v2"},
		PublicSyslogFQDN:   "127.1.1.1",
		TinkServerTLS:      false,
		TinkServerGRPCAddr: "127.0.0.1:42113",
		StaticIPXEEnabled:  true,
	}
	hf := h.HandlerFunc()
	writer := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auto.ipxe", nil)
	hf(writer, req)
	if writer.Code != 200 {
		t.Errorf("expected status code 200, got %d", writer.Code)
	}
	if diff := cmp.Diff(writer.Body.String(), want); diff != "" {
		t.Fatalf("expected custom script, got %s", diff)
	}
}
