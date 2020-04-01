package custom_ipxe

import (
	"strings"

	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

func init() {
	job.RegisterSlug("custom_ipxe", bootScript)
}

func bootScript(j job.Job, s *ipxe.Script) {
	s.PhoneHome("provisioning.104.01")
	s.Set("packet_facility", j.FacilityCode())
	s.Set("packet_plan", j.PlanSlug())

	if strings.HasPrefix(j.UserData(), "#!ipxe") {
		s.AppendString(strings.TrimPrefix(j.UserData(), "#!ipxe"))
	} else {
		s.Chain(j.IPXEScriptURL())
	}
}
