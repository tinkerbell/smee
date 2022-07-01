package ignition

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

type Config struct {
	Systemd SystemdUnits `json:"systemd,omitempty"`
	Network NetworkUnits `json:"networkd,omitempty"`
	Storage *Storage     `json:"storage,omitempty"`
	Passwd  *Passwd      `json:"passwd,omitempty"`
}

func (c *Config) Render(w io.Writer) error {
	v := struct {
		Version int `json:"ignitionVersion"`
		*Config
	}{Version: 1, Config: c}

	b, err := json.Marshal(&v)
	if err != nil {
		return errors.Wrap(err, "marshaling ignition config as json")
	}

	_, err = w.Write(b)
	if err != nil {
		return errors.Wrap(err, "writing ignition config")
	}

	return nil
}
