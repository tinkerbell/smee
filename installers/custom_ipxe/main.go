package custom_ipxe

import (
	"context"
	"strings"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
	"github.com/tinkerbell/boots/packet"
)

func init() {
	job.RegisterInstaller("custom_ipxe", ipxeScript)
	job.RegisterSlug("custom_ipxe", ipxeScript)
}

func ipxeScript(ctx context.Context, j job.Job, s *ipxe.Script) {
	logger := j.Logger.With("installer", "custom_ipxe")

	var cfg *packet.InstallerData

	if j.OperatingSystem().Installer == "custom_ipxe" {
		cfg = j.OperatingSystem().InstallerData
		if cfg == nil {
			s.Echo("Installer data not provided")
			s.Shell()
			logger.Error(ErrEmptyIpxeConfig, "installer data not provided")

			return
		}
	} else if strings.HasPrefix(j.UserData(), "#!ipxe") {
		cfg = &packet.InstallerData{Script: j.UserData()}
	} else if j.IPXEScriptURL() != "" {
		cfg = &packet.InstallerData{Chain: j.IPXEScriptURL()}
	} else {
		s.Echo("Unknown ipxe configuration")
		s.Shell()
		logger.Error(ErrEmptyIpxeConfig, "unknown ipxe configuration")

		return
	}

	ipxeScriptFromConfig(logger, cfg, j, s)
}

func ipxeScriptFromConfig(logger log.Logger, cfg *packet.InstallerData, j job.Job, s *ipxe.Script) {
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
