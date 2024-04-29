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
syslog_host={{ .SyslogHost }} grpc_authority={{ .TinkGRPCAuthority }} tinkerbell_tls={{ .TinkerbellTLS }} {{- range .ExtraKernelParams}} {{.}} {{- end}} \
worker_id=${mac} hw_addr=${mac} initrd=initramfs-${arch} console=tty1 console=tty2 console=tty3 console=tty4 console=tty5 console=ttyS3,9600 console=ttyS2,9600 console=tty6 console=ttyS1,9600 console=hvc0 console=ttyAMA0 console=ttyAMA1 console=tty0 console=ttyS0,9600
# linuxkit.unified_cgroup_hierarchy=1
initrd ${download-url}/initramfs-${arch}

boot
`
