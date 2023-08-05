package script

// HookScript is the default iPXE script for loading Hook.
var HookScript = `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...
{{- if .TraceID }}
echo Debug TraceID: {{ .TraceID }}
{{- end }}

set arch {{ .Arch }}
set download-url {{ .DownloadURL }}

kernel ${download-url}/vmlinuz-${arch} {{- if ne .VLANID "" }} vlan_id={{ .VLANID }} {{- end }} {{- range .ExtraKernelParams}} {{.}} {{- end}} \
facility={{ .Facility }} syslog_host={{ .SyslogHost }} grpc_authority={{ .TinkGRPCAuthority }} tinkerbell_tls={{ .TinkerbellTLS }} worker_id={{ .WorkerID }} hw_addr={{ .HWAddr }} \
modules=loop,squashfs,sd-mod,usb-storage intel_iommu=on iommu=pt initrd=initramfs-${arch} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`

// Hook holds the values used to generate the iPXE script that loads the Hook OS.
type Hook struct {
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
