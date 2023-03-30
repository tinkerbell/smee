package dhcp

import (
	"fmt"
	"strconv"

	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/tinkerbell/boots/conf"
)

const (
	EncapsulatedOptions = 175 // ipxe-encap-opts: encapsulate ipxe

	OptionPriority        = 1   // ipxe.priority: int8
	OptionKeepSAN         = 8   // ipxe.keep-san: uint8
	OptionSkipSANBoot     = 9   // ipxe.skip-san-boot: uint8
	OptionUserClass       = 77  // ipxe.user-class: string
	OptionSyslogs         = 85  // ipxe.syslogs: string
	OptionCertificate     = 91  // ipxe.cert: string
	OptionPrivateKey      = 92  // ipxe.privkey: string
	OptionCrossCert       = 93  // ipxe.crosscert: string
	OptionNoPXEDHCP       = 176 // ipxe.no-pxedhcp: uint8
	OptionBusID           = 177 // ipxe.bus-id: string
	OptionBIOSDrive       = 189 // ipxe.bios-drive: uint8
	OptionUsername        = 190 // ipxe.username: string
	OptionPassword        = 191 // ipxe.password: string
	OptionReverseUsername = 192 // ipxe.reverse-username: string
	OptionReversePassword = 193 // ipxe.reverse-password: string
	OptionInitiatorIQN    = 203 // iscsi-initiator-iqn: string
	OptionVersion         = 235 // ipxe.version: string

	FeaturePXEXT     = 16 // ipxe.pxeext: uint8
	FeatureISCSI     = 17 // ipxe.iscsi: uint8
	FeatureAoE       = 18 // ipxe.aoe: uint8
	FeatureHTTP      = 19 // ipxe.http: uint8
	FeatureHTTPS     = 20 // ipxe.https: uint8
	FeatureTFTP      = 21 // ipxe.tftp: uint8
	FeatureFTP       = 22 // ipxe.ftp: uint8
	FeatureDNS       = 23 // ipxe.dns: uint8
	FeatureBzImage   = 24 // ipxe.bzimage: uint8
	FeatureMultiboot = 25 // ipxe.multiboot: uint8
	FeatureSLAM      = 26 // ipxe.slam: uint8
	FeatureSRP       = 27 // ipxe.srp: uint8
	FeatureNBI       = 32 // ipxe.nbi: uint8
	FeaturePXE       = 33 // ipxe.pxe: uint8
	FeatureELF       = 34 // ipxe.elf: uint8
	FeatureCOMBOOT   = 35 // ipxe.comboot: uint8
	FeatureEFI       = 36 // ipxe.efi: uint8
	FeatureFCoE      = 37 // ipxe.fcoe: uint8
	FeatureVLAN      = 38 // ipxe.vlan: uint8
	FeatureMenu      = 39 // ipxe.menu: uint8
	FeatureSDI       = 40 // ipxe.sdi: uint8
	FeatureNFS       = 41 // ipxe.nfs: uint8
)

func init() {
	dhcp4.SetOptionFormatter(EncapsulatedOptions, func(b []byte) []interface{} {
		return formatOptions(parseOptions(b))
	})
}

var encapOptions = dhcp4.OptionMap{
	OptionNoPXEDHCP: []byte{1}, // ipxe.no-pxedhcp: Attempt to tell iPXE to boot faster.
}.Serialize()

func Setup(rep *dhcp4.Packet) {
	rep.SetOption(EncapsulatedOptions, encapOptions)
	rep.SetIP(dhcp4.OptionLogServer, conf.PublicSyslogIPv4) // Have iPXE send syslog to me.
}

// IsTinkerbellIPXE returns bool depending on if the DHCP request originated from an iPXE binary with its user-class (opt 77) set to "Tinkerbell"
// github.com/tinkerbell/boots-ipxe builds the iPXE binary with the user-class set to "Tinkerbell"
// https://ipxe.org/appnote/userclass
func IsTinkerbellIPXE(req *dhcp4.Packet) bool {
	uc, _ := req.GetOption(dhcp4.OptionUserClass)

	return string(uc) == "Tinkerbell"
}

// isIPXE returns bool depending on if the request originated with a version of iPXE.
func isIPXE(req *dhcp4.Packet) bool {
	if om := getEncapsulatedOptions(req); om != nil && hasFeature(om, FeatureHTTP) {
		return true
	}

	return false
}

func formatOptions(opts dhcp4.OptionMap) []interface{} {
	if opts == nil {
		return nil
	}

	fields := make([]interface{}, 0, 2*len(opts))
	for k, v := range opts {
		info, ok := options[k]
		if !ok {
			info.Name = fmt.Sprintf("ipxe.option(%d)", k)
			info.Format = formatOption
		}
		k, v := info.Format(&info, v)
		fields = append(fields, k, v)
	}

	return fields
}

func getEncapsulatedOptions(opts dhcp4.OptionGetter) dhcp4.OptionMap {
	if v, ok := opts.GetString(dhcp4.OptionUserClass); ok && v != "iPXE" {
		return nil
	}
	if x, ok := opts.GetOption(EncapsulatedOptions); ok {
		return parseOptions(x)
	}

	return nil
}

func hasFeature(opts dhcp4.OptionGetter, feature dhcp4.Option) bool {
	if opts == nil {
		return false
	}
	v, ok := opts.GetUint8(feature)

	return ok && v == 1
}

func parseOptions(b []byte) dhcp4.OptionMap {
	nested := make(dhcp4.OptionMap)
	if err := nested.Deserialize(b, &dhcp4.OptionMapDeserializeOptions{IgnoreMissingEndTag: true}); err != nil {
		// clog.Warning(err)
		return nil
	}

	return nested
}

type optionInfo struct {
	Name   string
	Type   string
	Format func(*optionInfo, []byte) (string, string)
}

func formatFeature(info *optionInfo, b []byte) (string, string) {
	if len(b) != 1 {
		return info.Name, string(b)
	}
	v := b[0]

	if info.Type == "bool" {
		if v == 1 {
			return info.Name, "true"
		}
	}

	return info.Name, fmt.Sprintf("%d", v)
}

func formatOption(info *optionInfo, b []byte) (string, string) {
	switch info.Type {
	case "string", "":
	case "hex":
		buf := make([]byte, 0, len(b)*2+len(b)-1)
		for i, c := range b {
			if i > 0 {
				buf = append(buf, ':')
			}
			buf = strconv.AppendUint(buf, uint64(c), 16)
		}
		b = buf
	case "bool":
		if len(b) == 1 && b[0] == 1 {
			return info.Name, "true"
		}

		fallthrough
	case "uint8", "int8":
		if len(b) == 1 {
			return info.Name, fmt.Sprintf("%d", b[0])
		}
	}

	return info.Name, fmt.Sprintf("%v", b)
}

func formatVersion(info *optionInfo, b []byte) (string, string) {
	if len(b) != 3 {
		return info.Name, string(b)
	}

	return info.Name, fmt.Sprintf("%d.%d.%d", b[0], b[1], b[2])
}

var options = map[dhcp4.Option]optionInfo{
	OptionPriority:        {"ipxe.priority", "int8", formatOption},
	OptionKeepSAN:         {"ipxe.keep-san", "bool", formatOption},
	OptionSkipSANBoot:     {"ipxe.skip-san-boot", "bool", formatOption},
	OptionSyslogs:         {"ipxe.syslogs", "string", formatOption},
	OptionCertificate:     {"ipxe.cert", "hex", formatOption},
	OptionPrivateKey:      {"ipxe.privkey", "hex", formatOption},
	OptionCrossCert:       {"ipxe.crosscert", "hex", formatOption},
	OptionNoPXEDHCP:       {"ipxe.no-pxedhcp", "bool", formatOption},
	OptionBusID:           {"ipxe.bus-id", "hex", formatOption},
	OptionBIOSDrive:       {"ipxe.bios-drive", "uint8", formatOption},
	OptionUsername:        {"ipxe.username", "string", formatOption},
	OptionPassword:        {"ipxe.password", "string", formatOption},
	OptionReverseUsername: {"ipxe.reverse-username", "string", formatOption},
	OptionReversePassword: {"ipxe.reverse-password", "string", formatOption},
	OptionVersion:         {"ipxe.version", "string", formatVersion},
	OptionInitiatorIQN:    {"iscsi-initiator-iqn", "string", formatOption},

	// Feature Indicators
	FeaturePXEXT:     {"ipxe.pxeext", "uint8", formatFeature},   // PXEXT
	FeatureISCSI:     {"ipxe.iscsi", "bool", formatFeature},     // iSCSI
	FeatureAoE:       {"ipxe.aoe", "bool", formatFeature},       // AoE
	FeatureHTTP:      {"ipxe.http", "bool", formatFeature},      // HTTP
	FeatureHTTPS:     {"ipxe.https", "bool", formatFeature},     // HTTPS
	FeatureTFTP:      {"ipxe.tftp", "bool", formatFeature},      // TFTP
	FeatureFTP:       {"ipxe.ftp", "bool", formatFeature},       // FTP
	FeatureDNS:       {"ipxe.dns", "bool", formatFeature},       // DNS
	FeatureBzImage:   {"ipxe.bzimage", "bool", formatFeature},   // bzImage
	FeatureMultiboot: {"ipxe.multiboot", "bool", formatFeature}, // MBOOT
	FeatureSLAM:      {"ipxe.slam", "bool", formatFeature},      // SLAM
	FeatureSRP:       {"ipxe.srp", "bool", formatFeature},       // SRP
	FeatureNBI:       {"ipxe.nbi", "bool", formatFeature},       // NBI
	FeaturePXE:       {"ipxe.pxe", "bool", formatFeature},       // PXE
	FeatureELF:       {"ipxe.elf", "bool", formatFeature},       // ELF
	FeatureCOMBOOT:   {"ipxe.comboot", "bool", formatFeature},   // COMBOOT
	FeatureEFI:       {"ipxe.efi", "bool", formatFeature},       // EFI
	FeatureFCoE:      {"ipxe.fcoe", "bool", formatFeature},      // FCoE
	FeatureVLAN:      {"ipxe.vlan", "bool", formatFeature},      // VLAN
	FeatureMenu:      {"ipxe.menu", "bool", formatFeature},      // Menu
	FeatureSDI:       {"ipxe.sdi", "bool", formatFeature},       // SDI
	FeatureNFS:       {"ipxe.nfs", "bool", formatFeature},       // NFS
}
