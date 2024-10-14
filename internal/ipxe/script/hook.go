package script

// HookScript is the default iPXE script for loading Hook.
var HookScript = `#!ipxe

echo Loading the Tinkerbell Hook iPXE script...
{{- if .TraceID }}
echo Debug TraceID: {{ .TraceID }}
{{- end }}

set arch {{ .Arch }}
set download-url {{ .DownloadURL }}
set kernel {{ if .Kernel }}{{ .Kernel }}{{ else }}vmlinuz-${arch}{{ end }}
set initrd {{ if .Initrd }}{{ .Initrd }}{{ else }}initramfs-${arch}{{ end }}
set retries:int32 {{ .Retries }}
set retry_delay:int32 {{ .RetryDelay }}

set idx:int32 0
:retry_kernel
kernel ${download-url}/${kernel} {{- if ne .VLANID "" }} vlan_id={{ .VLANID }} {{- end }} {{- range .ExtraKernelParams}} {{.}} {{- end}} \
facility={{ .Facility }} syslog_host={{ .SyslogHost }} grpc_authority={{ .TinkGRPCAuthority }} tinkerbell_tls={{ .TinkerbellTLS }} tinkerbell_insecure_tls={{ .TinkerbellInsecureTLS }} worker_id={{ .WorkerID }} hw_addr={{ .HWAddr }} \
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

// Hook holds the values used to generate the iPXE script that loads the Hook OS.
type Hook struct {
	Arch                  string   // example x86_64
	Console               string   // example ttyS1,115200
	DownloadURL           string   // example https://location:8080/to/kernel/and/initrd
	ExtraKernelParams     []string // example tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0
	Facility              string
	HWAddr                string // example 3c:ec:ef:4c:4f:54
	SyslogHost            string
	TinkerbellTLS         bool
	TinkerbellInsecureTLS bool
	TinkGRPCAuthority     string // example 192.168.2.111:42113
	TraceID               string
	VLANID                string // string number between 1-4095
	WorkerID              string // example 3c:ec:ef:4c:4f:54 or worker1
	Retries               int    // number of retries to attempt when fetching kernel and initrd files
	RetryDelay            int    // number of seconds to wait between retries
	Kernel                string // name of the kernel file
	Initrd                string // name of the initrd file
}
