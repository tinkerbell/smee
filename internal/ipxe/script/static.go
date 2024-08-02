package script

// StaticScript is the iPXE script used when in the auto-proxy mode.
// It is built to be generic enough for all hardware to use.
var StaticScript = `#!ipxe

echo Loading the static Tinkerbell iPXE script...

set arch ${buildarch}
# Tinkerbell only supports 64 bit archectures.
# The build architecture does not necessarily represent the architecture of the machine on which iPXE is running.
# https://ipxe.org/cfg/buildarch
iseq ${arch} i386 && set arch x86_64 ||
iseq ${arch} arm32 && set arch aarch64 ||
iseq ${arch} arm64 && set arch aarch64 ||
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
console=tty1 console=tty2 console=ttyAMA0,115200 console=ttyAMA1,115200 console=ttyS0,115200 console=ttyS1,115200 {{- range .ExtraKernelParams}} {{.}} {{- end}} \
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
