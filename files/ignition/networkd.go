package ignition

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/files/unit"
)

type NetworkUnits []*unit.Unit

func (us *NetworkUnits) Add(name string) *unit.Unit {
	u := unit.New(name)
	*us = append(*us, u)

	return u
}

func (us *NetworkUnits) Append(u *unit.Unit) {
	*us = append(*us, u)
}

func (us NetworkUnits) MarshalJSON() ([]byte, error) {
	if len(us) == 0 {
		return nil, nil
	}
	var v struct {
		Units []*unit.Unit `json:"units"`
	}
	v.Units = us

	b, err := json.Marshal(&v)

	return b, errors.Wrap(err, "marshaling network unit as json")
}
