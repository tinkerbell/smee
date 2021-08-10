package custom_ipxe

import (
	"encoding/json"
	"strings"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

func init() {
	job.RegisterInstaller("ipxe", ipxeScript)
	job.RegisterSlug("custom_ipxe", ipxeScript)
}

func ipxeScript(j job.Job, s *ipxe.Script) {
	logger := installers.Logger("ipxe")
	if j.InstanceID() != "" {
		logger = logger.With("instance.id", j.InstanceID())
	}

	var cfg *Config
	var err error

	if j.OperatingSystem().Installer == "ipxe" {
		cfg, err = ipxeConfigFromJob(j)
		if err != nil {
			s.Echo("Failed to decode installer data")
			s.Shell()
			logger.Error(err, "decoding installer data")

			return
		}
	} else {
		cfg = &Config{}

		if strings.HasPrefix(j.UserData(), "#!ipxe") {
			cfg.Script = j.UserData()
		} else {
			cfg.Chain = j.IPXEScriptURL()
		}
	}

	IpxeScriptFromConfig(logger, cfg, j, s)
}

func IpxeScriptFromConfig(logger log.Logger, cfg *Config, j job.Job, s *ipxe.Script) {
	if err := cfg.validate(); err != nil {
		s.Echo("Invalid ipxe configuration")
		s.Shell()
		logger.Error(err, "validating ipxe config")

		return
	}

	s.PhoneHome("provisioning.104.01")
	s.Set("packet_facility", j.FacilityCode())
	s.Set("packet_plan", j.PlanSlug())

	if cfg.Chain != "" {
		s.Chain(cfg.Chain)
	} else if cfg.Script != "" {
		s.AppendString(strings.TrimPrefix(cfg.Script, "#!ipxe"))
	} else {
		s.Echo("Unknown ipxe config path")
		s.Shell()
	}
}

func ipxeConfigFromJob(j job.Job) (*Config, error) {
	data := j.OperatingSystem().InstallerData

	cfg := &Config{}

	err := json.NewDecoder(strings.NewReader(data)).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

type Config struct {
	Chain  string `json:"chain,omitempty"`
	Script string `json:"script,omitempty"`
}

func (c *Config) validate() error {
	if c.Chain == "" && c.Script == "" {
		return ErrEmptyIpxeConfig
	}

	return nil
}
