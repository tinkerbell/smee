package conf

import (
	"fmt"
	"strings"

	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
)

const defaultFacility = "ewr1"

var (
	FacilityCode  = env.Get("FACILITY_CODE", defaultFacility)
	MirrorBaseURL = mustBuildMirrorBaseURL()
)

func mustBuildMirrorBaseURL() string {
	s, err := buildMirrorBaseURL()
	if s == "" {
		panic(fmt.Sprintf(`conf: MirrorBaseURL: %q: %s\n`, s, err))
	}

	return s
}

func buildMirrorBaseURL() (string, error) {
	host := env.URL("MIRROR_BASE_URL", "http://install."+FacilityCode+".packet.net")
	if host.Scheme != "http" && host.Scheme != "https" {
		return "", errors.Errorf("parsed scheme is neither http nor https: %s", host.Scheme)
	}
	if host.RawQuery != "" {
		return "", errors.Errorf("parsed URL should not contain a query: %s", host.RawQuery)
	}
	if host.Fragment != "" {
		return "", errors.Errorf("parsed URL should not contain a fragment: %s", host.Fragment)
	}

	return strings.TrimRight(host.String(), "/"), nil
}
