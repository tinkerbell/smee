package ipxe

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type auto struct {
	Arch              string   // example x86_64
	Console           string   // example ttyS1,115200
	DownloadURL       string   // example https://location:8080/to/kernel/and/initrd
	ExtraKernelParams []string // example tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0
	Facility          string
	HWAddr            string // example 3c:ec:ef:4c:4f:54
	SyslogHost        string
	TinkerbellTLS     bool
	TinkGRPCAuthority string // example 192.168.2.111:42113
	TraceID           string
	VLANID            string // string number between 1-4095
	WorkerID          string // example 3c:ec:ef:4c:4f:54 or worker1
}

func TestGenerateTemplate(t *testing.T) {
	tests := map[string]struct {
		a       auto
		script  string
		want    string
		wantErr bool
	}{
		"no vlan": {
			a: auto{
				Arch:              "x86_64",
				TinkGRPCAuthority: "1.2.3.4:42113",
				TinkerbellTLS:     false,
				WorkerID:          "3c:ec:ef:4c:4f:54",
				SyslogHost:        "1.2.3.4",
				DownloadURL:       "http://location:8080/to/kernel/and/initrd",
				Facility:          "onprem",
				ExtraKernelParams: []string{"tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0", "tinkerbell=packet"},
				HWAddr:            "3c:ec:ef:4c:4f:54",
			},
			script: AutoScript,
			want: `#!ipxe

echo Loading the Tinkerbell iPXE script...

set arch x86_64
set download-url http://location:8080/to/kernel/and/initrd

kernel ${download-url}/vmlinuz-${arch} ip=dhcp tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 tinkerbell=packet \
facility=onprem syslog_host=1.2.3.4 grpc_authority=1.2.3.4:42113 tinkerbell_tls=false worker_id=3c:ec:ef:4c:4f:54 hw_addr=3c:ec:ef:4c:4f:54 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`,
		},
		"with vlan": {
			a: auto{
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
			},
			script: AutoScript,
			want: `#!ipxe

echo Loading the Tinkerbell iPXE script...

set arch x86_64
set download-url http://location:8080/to/kernel/and/initrd

kernel ${download-url}/vmlinuz-${arch} vlan_id=16 tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 tinkerbell=packet \
facility=onprem syslog_host=1.2.3.4 grpc_authority=1.2.3.4:42113 tinkerbell_tls=false worker_id=3c:ec:ef:4c:4f:54 hw_addr=3c:ec:ef:4c:4f:54 \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`,
		},
		"parse error": {
			a:       auto{},
			script:  "bad {{ }",
			wantErr: true,
		},
		"execute error": {
			a:       auto{},
			script:  "{{ .A }}",
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GenerateTemplate(tt.a, tt.script)
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
