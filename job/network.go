package job

import "github.com/tinkerbell/boots/client"

func (j Job) BondingMode() client.BondingMode {
	return j.hardware.HardwareBondingMode()
}
