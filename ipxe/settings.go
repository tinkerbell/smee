package ipxe

import "fmt"

const (
	AssetTag      = "${asset}"        // Unfilled
	BoardSerial   = "${board-serial}" // Server Serial Number
	Manufacturer  = "${manufacturer}" // Manufacturer Name
	ModelNumber   = "${product}"      // Chassis Model Number
	ChassisSerial = "${serial}"       // Chassis Serial Number
	UUID          = "${uuid}"         // 00000000-0000-0000-0000-${mac}
)

type smbiosTag struct {
	Instance byte
	Type     byte
	Offset   byte
	Len      byte
}

func (t smbiosTag) String() string {
	return fmt.Sprintf("smbios/%d.%d.%d.%d", t.Instance, t.Type, t.Offset, t.Len)
}
