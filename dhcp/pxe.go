package dhcp

import (
	"context"
	"net"
	"strings"

	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

// from https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml
var procArchTypes = []string{
	"x86 BIOS",
	"NEC/PC98 (DEPRECATED)",
	"Itanium",
	"DEC Alpha (DEPRECATED)",
	"Arc x86 (DEPRECATED)",
	"Intel Lean Client (DEPRECATED)",
	"x86 UEFI",
	"x64 UEFI",
	"EFI Xscale (DEPRECATED)",
	"EBC",
	"ARM 32-bit UEFI",
	"ARM 64-bit UEFI",
	"PowerPC Open Firmware",
	"PowerPC ePAPR",
	"POWER OPAL v3",
	"x86 uefi boot from http",
	"x64 uefi boot from http",
	"ebc boot from http",
	"arm uefi 32 boot from http",
	"arm uefi 64 boot from http",
	"pc/at bios boot from http",
	"arm 32 uboot",
	"arm 64 uboot",
	"arm uboot 32 boot from http",
	"arm uboot 64 boot from http",
	"RISC-V 32-bit UEFI",
	"RISC-V 32-bit UEFI boot from http",
	"RISC-V 64-bit UEFI",
	"RISC-V 64-bit UEFI boot from http",
	"RISC-V 128-bit UEFI",
	"RISC-V 128-bit UEFI boot from http",
	"s390 Basic",
	"s390 Extended",
}

func ProcessorArchType(req *dhcp4.Packet) string {
	v, ok := req.GetUint16(dhcp4.OptionClientSystem)
	if !ok || int(v) >= len(procArchTypes) {
		return ""
	}

	return procArchTypes[v]
}

func Arch(req *dhcp4.Packet) string {
	arch := ProcessorArchType(req)
	switch arch {
	case "x86 BIOS", "x64 UEFI":
		return "x86_64"
	case "ARM 64-bit UEFI":
		return "aarch64"
	default:
		return arch
	}
}

func IsARM(req *dhcp4.Packet) bool {
	return strings.Contains(strings.ToLower(ProcessorArchType(req)), "arm")
}

func IsUEFI(req *dhcp4.Packet) bool {
	return strings.Contains(strings.ToLower(ProcessorArchType(req)), "uefi")
}

func IsPXE(req *dhcp4.Packet) bool {
	_, ok := req.GetOption(dhcp4.OptionUUIDGUID)
	if ok {
		return ok
	}
	class, ok := req.GetString(dhcp4.OptionClassID)

	return ok && strings.HasPrefix(class, "PXEClient")
}

func SetupPXE(ctx context.Context, rep, req *dhcp4.Packet) bool {
	if !copyGUID(rep, req) {
		if class, ok := req.GetString(dhcp4.OptionClassID); !ok || !strings.HasPrefix(class, "PXEClient") {
			return false // not a PXE client
		}
		dhcplog.With("mac", req.GetCHAddr(), "xid", req.GetXID()).Info("no client GUID provided")
	}

	/*
		Intel's Preboot Execution Environment (PXE) Specification (1999):

		Options 1-63 are reserved for PXE, we use option 6 with value 0x8:

			These options control the type of boot server discovery mechanisms used
			by clients. Clients must use discovery methods in this order:
			[options 1 & 2 omitted bc they're irrelevant to us]

			3. Unicast. If a Boot Server list is available, (PXE_BOOT_SERVERS,
			   Option #43 tag #8). If PXE_DISCOVERY_CONTROL bit 2 is set, the client
			   may still use multicast and broadcast discovery (if it is permitted
			   by bits 0 and 1); but the client may only accept replies from servers
			   that are identified in the PXE_BOOT_SERVERS option.

		Options 64-127 are "boot server specific" so that's why we put traceparent
		propagation in opt43/slot69.
	*/

	pxeVendorOptions := dhcp4.OptionMap{
		6:  []byte{0x8}, // PXE_DISCOVERY_CONTROL: Attempt to tell PXE to boot faster.
		69: binaryTpFromContext(ctx),
	}.Serialize()

	rep.SetOption(dhcp4.OptionVendorSpecific, pxeVendorOptions)

	return true
}

func SetFilename(rep *dhcp4.Packet, filename string, nextServer net.IP, pxeClient bool) {
	file := rep.File()
	if len(filename) > len(file) {
		err := errors.New("filename too long, would be truncated")
		// req CHaddr and XID == req's
		dhcplog.With("mac", rep.GetCHAddr(), "xid", rep.GetXID(), "filename", filename).Fatal(err)
	}
	if pxeClient {
		rep.SetString(dhcp4.OptionClassID, "PXEClient")
	}
	rep.SetSIAddr(nextServer) // next-server: IP address of the TFTP/HTTP Server.
	copy(file, filename)      // filename: Executable (or iPXE script) to boot from.
}

func copyGUID(rep, req *dhcp4.Packet) bool {
	if guid, ok := req.GetOption(dhcp4.OptionUUIDGUID); ok {
		// only accepts 16-byte client GUIDs and type 0x0000
		// e.g. dhcpcd on Linux uses 36 bytes and type 0x00ff so it will be ignored
		if len(guid) != 17 || guid[0] != 0 {
			dhcplog.With("guid", guid, "mac", req.GetCHAddr(), "xid", req.GetXID()).Error(errors.New("unsupported or malformed client GUID"))
		} else {
			rep.SetOption(dhcp4.OptionUUIDGUID, guid)

			return true
		}
	}

	return false
}

// binaryTpFromContext extracts the binary trace id, span id, and trace flags
// from the running span in ctx and returns a 26 byte []byte with the traceparent
// encoded and ready to pass in opt43
// see test/test-boots.sh for how to decode tp with busybox udhcpc & cut(1)
func binaryTpFromContext(ctx context.Context) []byte {
	sc := trace.SpanContextFromContext(ctx)
	tpBytes := make([]byte, 0, 26)

	// the otel spec says 16 bytes for trace id and 8 for spans are good enough
	// for everyone copy them into a []byte that we can deliver over option43
	tid := [16]byte(sc.TraceID()) // type TraceID [16]byte
	sid := [8]byte(sc.SpanID())   // type SpanID [8]byte

	tpBytes = append(tpBytes, 0x00)      // traceparent version
	tpBytes = append(tpBytes, tid[:]...) // trace id
	tpBytes = append(tpBytes, sid[:]...) // span id
	if sc.IsSampled() {
		tpBytes = append(tpBytes, 0x01) // trace flags
	} else {
		tpBytes = append(tpBytes, 0x00)
	}

	return tpBytes
}
