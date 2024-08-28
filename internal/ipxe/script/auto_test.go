package script

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateTemplate(t *testing.T) {
	tests := map[string]struct {
		h       Hook
		script  string
		want    string
		wantErr bool
	}{
		"no vlan": {
			h: Hook{
				Arch:              "x86_64",
				TinkGRPCAuthority: "1.2.3.4:42113",
				TinkerbellTLS:     false,
				WorkerID:          "3c:ec:ef:4c:4f:54",
				SyslogHost:        "1.2.3.4",
				DownloadURL:       "http://location:8080/to/kernel/and/initrd",
				Facility:          "onprem",
				ExtraKernelParams: []string{"tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0", "tinkerbell=packet"},
				HWAddr:            "3c:ec:ef:4c:4f:54",
				Retries:           10,
				RetryDelay:        3,
			},
			script: HookScript,
			want: `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...

set arch x86_64
set download-url http://location:8080/to/kernel/and/initrd
set retries:int32 10
set retry_delay:int32 3

set idx:int32 0
:retry_kernel
kernel ${download-url}/vmlinuz-${arch} tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 tinkerbell=packet \
facility=onprem syslog_host=1.2.3.4 grpc_authority=1.2.3.4:42113 tinkerbell_tls=false tinkerbell_insecure_tls=false worker_id=3c:ec:ef:4c:4f:54 hw_addr=3c:ec:ef:4c:4f:54 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200 && goto download_initrd || iseq ${idx} ${retries} && goto kernel-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_kernel

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
`,
		},
		"with vlan": {
			h: Hook{
				Arch:              "x86_64",
				TinkGRPCAuthority: "1.2.3.4:42113",
				TinkerbellTLS:     false,
				WorkerID:          "3c:ec:ef:4c:4f:54",
				SyslogHost:        "1.2.3.4",
				DownloadURL:       "http://location:8080/to/kernel/and/initrd",
				Facility:          "onprem",
				ExtraKernelParams: []string{"tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0", "tinkerbell=packet"},
				HWAddr:            "3c:ec:ef:4c:4f:54",
				VLANID:            "16",
				Retries:           10,
				RetryDelay:        3,
			},
			script: HookScript,
			want: `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...

set arch x86_64
set download-url http://location:8080/to/kernel/and/initrd
set retries:int32 10
set retry_delay:int32 3

set idx:int32 0
:retry_kernel
kernel ${download-url}/vmlinuz-${arch} vlan_id=16 tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 tinkerbell=packet \
facility=onprem syslog_host=1.2.3.4 grpc_authority=1.2.3.4:42113 tinkerbell_tls=false tinkerbell_insecure_tls=false worker_id=3c:ec:ef:4c:4f:54 hw_addr=3c:ec:ef:4c:4f:54 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200 && goto download_initrd || iseq ${idx} ${retries} && goto kernel-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_kernel

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
`,
		},
		"parse error": {
			h:       Hook{},
			script:  "bad {{ }",
			wantErr: true,
		},
		"execute error": {
			h:       Hook{},
			script:  "{{ .A }}",
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GenerateTemplate(tt.h, tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("Auto.autoDotIPXE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Auto.autoDotIPXE() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
