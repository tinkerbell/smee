package custom_ipxe

import (
	"strings"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/packet"
)

func init() {
	job.RegisterInstaller("ipxe", ipxeScript)
	job.RegisterSlug("custom_ipxe", ipxeScript)
}

func ipxeScript(j job.Job, s *ipxe.Script) {
	logger := installers.Logger("custom_ipxe")
	if j.InstanceID() != "" {
		logger = logger.With("instance.id", j.InstanceID())
	}

	var cfg *packet.InstallerData
	var err error

	if j.OperatingSystem().Installer == "ipxe" {
		cfg = j.OperatingSystem().InstallerData
		if cfg == nil {
			s.Echo("Installer data not provided")
			s.Shell()
			logger.Error(err, "empty installer data")

			return
		}
	} else {
		if strings.HasPrefix(j.UserData(), "#!ipxe") {
			cfg = &packet.InstallerData{Script: j.UserData()}
		} else {
			cfg = &packet.InstallerData{Chain: j.IPXEScriptURL()}
		}
	}

	IpxeScriptFromConfig(logger, cfg, j, s)
}

func IpxeScriptFromConfig(logger log.Logger, cfg *packet.InstallerData, j job.Job, s *ipxe.Script) {
	if err := validateConfig(cfg); err != nil {
		s.Echo(err.Error())
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
	}
}

func validateConfig(c *packet.InstallerData) error {
	if c.Chain == "" && c.Script == "" {
		return ErrEmptyIpxeConfig
	}

	return nil
}
