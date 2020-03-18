package job

import "github.com/packethost/boots/packet"

func (j Job) BondingMode() packet.BondingMode {
	return j.hardware.BondingMode
}
