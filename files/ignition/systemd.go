package ignition

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/files/unit"
)

type SystemdUnit struct {
	unit.Unit
	Enabled bool         `json:"enable,omitempty"`
	Masked  bool         `json:"mask,omitempty"`
	Dropins []*unit.Unit `json:"dropins,omitempty"`
}

func NewSystemdUnit(name string) *SystemdUnit {
	u := new(SystemdUnit)
	u.Name = name

	return u
}

func (u *SystemdUnit) AddDropin(name string) *unit.Unit {
	d := unit.New(name)
	u.Dropins = append(u.Dropins, d)

	return d
}

func (u *SystemdUnit) Enable() *SystemdUnit {
	u.Enabled = true

	return u
}

func (u *SystemdUnit) Mask() *SystemdUnit {
	u.Masked = true

	return u
}

type SystemdUnits []*SystemdUnit

func (us *SystemdUnits) Add(name string) *SystemdUnit {
	u := NewSystemdUnit(name)
	*us = append(*us, u)

	return u
}

func (us SystemdUnits) MarshalJSON() ([]byte, error) {
	if len(us) == 0 {
		return nil, nil
	}
	var v struct {
		Units []*SystemdUnit `json:"units"`
	}
	v.Units = us

	b, err := json.Marshal(&v)

	return b, errors.Wrap(err, "marshalling unit as json")
}
