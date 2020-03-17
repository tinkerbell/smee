package job

import "github.com/packethost/tinkerbell/packet"

func (j Job) BondingMode() packet.BondingMode {
	return j.hardware.BondingMode
}
