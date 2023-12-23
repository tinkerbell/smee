package dhcp

import (
	"fmt"
	"strings"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
)

const (
	PXEClient  ClientType = "PXEClient"
	HTTPClient ClientType = "HTTPClient"
)

// known user-class types. must correspond to DHCP option 77 - User-Class
// https://www.rfc-editor.org/rfc/rfc3004.html
const (
	// If the client has had iPXE burned into its ROM (or is a VM
	// that uses iPXE as the PXE "ROM"), special handling is
	// needed because in this mode the client is using iPXE native
	// drivers and chainloading to a UNDI stack won't work.
	IPXE UserClass = "iPXE"
	// If the client identifies as "Tinkerbell", we've already
	// chainloaded this client to the full-featured copy of iPXE
	// we supply. We have to distinguish this case so we don't
	// loop on the chainload step.
	Tinkerbell UserClass = "Tinkerbell"
)

// UserClass is DHCP option 77 (https://www.rfc-editor.org/rfc/rfc3004.html).
type UserClass string

// ClientType is from DHCP option 60. Normally only PXEClient or HTTPClient.
type ClientType string

// ArchToBootFile maps supported hardware PXE architectures types to iPXE binary files.
var ArchToBootFile = map[iana.Arch]string{
	iana.INTEL_X86PC:       "undionly.kpxe",
	iana.NEC_PC98:          "undionly.kpxe",
	iana.EFI_ITANIUM:       "undionly.kpxe",
	iana.DEC_ALPHA:         "undionly.kpxe",
	iana.ARC_X86:           "undionly.kpxe",
	iana.INTEL_LEAN_CLIENT: "undionly.kpxe",
	iana.EFI_IA32:          "ipxe.efi",
	iana.EFI_X86_64:        "ipxe.efi",
	iana.EFI_XSCALE:        "ipxe.efi",
	iana.EFI_BC:            "ipxe.efi",
	iana.EFI_ARM32:         "snp.efi",
	iana.EFI_ARM64:         "snp.efi",
	iana.EFI_X86_HTTP:      "ipxe.efi",
	iana.EFI_X86_64_HTTP:   "ipxe.efi",
	iana.EFI_ARM32_HTTP:    "snp.efi",
	iana.EFI_ARM64_HTTP:    "snp.efi",
	iana.Arch(41):          "snp.efi", // arm rpiboot: https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml#processor-architecture
}

// ErrUnknownArch is used when the PXE client request is from an unknown architecture.
var ErrUnknownArch = fmt.Errorf("could not determine client architecture from option 93")

// Arch returns the Arch of the client pulled from DHCP option 93.
func Arch(d *dhcpv4.DHCPv4) iana.Arch {
	// get option 93 ; arch
	fwt := d.ClientArch()
	if len(fwt) == 0 {
		return iana.Arch(255) // unknown arch
	}
	var archKnown bool
	var a iana.Arch
	for _, elem := range fwt {
		if !strings.Contains(elem.String(), "unknown") {
			archKnown = true
			// Basic architecture identification, based purely on
			// the PXE architecture option.
			// https://www.iana.org/assignments/dhcpv6-parameters/dhcpv6-parameters.xhtml#processor-architecture
			a = elem
			break
		}
	}
	if !archKnown {
		return iana.Arch(255) // unknown arch
	}

	return a
}

// String function for clientType.
func (c ClientType) String() string {
	return string(c)
}

// String function for UserClass.
func (u UserClass) String() string {
	return string(u)
}

// IsNetbootClient returns true if the client is a valid netboot client.
//
// A valid netboot client will have the following in its DHCP request:
// 1. is a DHCP discovery/request message type.
// 2. option 93 is set.
// 3. option 94 is set.
// 4. option 97 is correct length.
// 5. option 60 is set with this format: "PXEClient:Arch:xxxxx:UNDI:yyyzzz" or "HTTPClient:Arch:xxxxx:UNDI:yyyzzz".
//
// See: http://www.pix.net/software/pxeboot/archive/pxespec.pdf
//
// See: https://www.rfc-editor.org/rfc/rfc4578.html
func IsNetbootClient(pkt *dhcpv4.DHCPv4) error {
	var err error
	// only response to DISCOVER and REQUEST packets
	if pkt.MessageType() != dhcpv4.MessageTypeDiscover && pkt.MessageType() != dhcpv4.MessageTypeRequest {
		// h.Log.Info("not a netboot client", "reason", "message type must be either Discover or Request", "mac", pkt.ClientHWAddr.String(), "message type", pkt.MessageType())
		err = wrapNonNil(err, "message type must be either Discover or Request")
	}
	// option 60 must be set
	if !pkt.Options.Has(dhcpv4.OptionClassIdentifier) {
		// h.Log.Info("not a netboot client", "reason", "option 60 not set", "mac", pkt.ClientHWAddr.String())
		err = wrapNonNil(err, "option 60 not set")
	}
	// option 60 must start with PXEClient or HTTPClient
	opt60 := pkt.GetOneOption(dhcpv4.OptionClassIdentifier)
	if !strings.HasPrefix(string(opt60), string(PXEClient)) && !strings.HasPrefix(string(opt60), string(HTTPClient)) {
		// h.Log.Info("not a netboot client", "reason", "option 60 not PXEClient or HTTPClient", "mac", pkt.ClientHWAddr.String(), "option 60", string(opt60))
		err = wrapNonNil(err, "option 60 not PXEClient or HTTPClient")
	}

	// option 93 must be set
	if !pkt.Options.Has(dhcpv4.OptionClientSystemArchitectureType) {
		// h.Log.Info("not a netboot client", "reason", "option 93 not set", "mac", pkt.ClientHWAddr.String())
		err = wrapNonNil(err, "option 93 not set")
	}

	// option 94 must be set
	if !pkt.Options.Has(dhcpv4.OptionClientNetworkInterfaceIdentifier) {
		// h.Log.Info("not a netboot client", "reason", "option 94 not set", "mac", pkt.ClientHWAddr.String())
		err = wrapNonNil(err, "option 94 not set")
	}

	// option 97 must be have correct length or not be set
	guid := pkt.GetOneOption(dhcpv4.OptionClientMachineIdentifier)
	switch len(guid) {
	case 0:
		// A missing GUID is invalid according to the spec, however
		// there are PXE ROMs in the wild that omit the GUID and still
		// expect to boot. The only thing we do with the GUID is
		// mirror it back to the client if it's there, so we might as
		// well accept these buggy ROMs.
	case 17:
		if guid[0] != 0 {
			err = wrapNonNil(err, "option 97 does not start with 0")
		}
	default:
		err = wrapNonNil(err, "option 97 has invalid length (must be 0 or 17)")
	}

	return err
}

func wrapNonNil(err error, format string, a ...any) error {
	if err == nil {
		return fmt.Errorf(format, a...)
	}

	return fmt.Errorf("%w: %w", err, fmt.Errorf(format, a...))
}
