package script

// AutoDiscoveryScript is the iPXE script used when in the auto discovery mode.
// This script will not do any hardware look ups. All hardware will get the same script.
var AutoDiscoveryScript = `#!ipxe

echo Loading the Tinkerbell Auto Discovery iPXE script...

set arch ${buildarch}
set download-url {{ .DownloadURL }}
set retries:int32 {{ .Retries }}
set retry_delay:int32 {{ .RetryDelay }}

set worker_id ${mac}
set grpc_authority {{ .TinkGRPCAuthority }}
set syslog_host {{ .SyslogHost }}
set tinkerbell_tls {{ .TinkerbellTLS }}

echo worker_id=${mac}
echo grpc_authority={{ .TinkGRPCAuthority }}
echo syslog_host={{ .SyslogHost }}
echo tinkerbell_tls={{ .TinkerbellTLS }}

set idx:int32 0
:retry_kernel
kernel ${download-url}/vmlinuz-${arch} \
syslog_host=${syslog_host} grpc_authority=${grpc_authority} tinkerbell_tls=${tinkerbell_tls} worker_id=${worker_id} hw_addr=${mac} \
console=tty1 console=tty2 console=ttyAMA0,115200 console=ttyAMA1,115200 console=ttyS0,115200 console=ttyS1,115200 \
intel_iommu=on iommu=pt {{- range .ExtraKernelParams}} {{.}} {{- end}} initrd=initramfs-${arch} && goto download_initrd || iseq ${idx} ${retries} && goto kernel-error || inc idx && echo retry in ${retry_delay} seconds ; sleep ${retry_delay} ; goto retry_kernel

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
