package script

// AutoDiscoveryScript is the iPXE script used when in the auto discovery mode.
// This script will not do any hardware look ups. All hardware will get the same script.
var AutoDiscoveryScript = `#!ipxe

echo Loading the Tinkerbell Auto Discovery iPXE script...

set arch ${buildarch}

set download-url {{ .DownloadURL }}

echo worker_id=${mac}
echo grpc_authority={{ .TinkGRPCAuthority }}
echo syslog_host={{ .SyslogHost }}
echo tinkerbell_tls={{ .TinkerbellTLS }}

kernel ${download-url}/vmlinuz-${arch} \
syslog_host={{ .SyslogHost }} grpc_authority={{ .TinkGRPCAuthority }} tinkerbell_tls={{ .TinkerbellTLS }} \
worker_id=${mac} hw_addr=${mac} initrd=initramfs-${arch} {{- range .ExtraKernelParams}} {{.}} {{- end}} console=tty0 console=ttyS1,115200

initrd ${download-url}/initramfs-${arch}

boot
`

type AutoDiscovery struct {
	DownloadURL       string   // example https://location:8080/to/kernel/and/initrd
	ExtraKernelParams []string // example tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0
	SyslogHost        string
	TinkerbellTLS     bool
	TinkGRPCAuthority string // example 192.168.2.111:42113
}
