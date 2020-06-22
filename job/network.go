package job

import "github.com/tinkerbell/boots/packet"

func (j Job) BondingMode() packet.BondingMode {
	return (*j.hardware).HardwareBondingMode()
}
