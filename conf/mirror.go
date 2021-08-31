package conf

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
)

const (
	defaultFacility   = "ewr1"
	defaultMirrorPath = "/misc/tinkerbell"
)

var (
	FacilityCode  = env.Get("FACILITY_CODE", defaultFacility)
	mirrorURL     = mustBuildMirrorURL()
	MirrorURL     = mirrorURL.String()
	MirrorHost    = mirrorURL.Host
	MirrorPath    = mirrorURL.Path
	MirrorBase    = strings.TrimSuffix(MirrorURL, MirrorPath)
	mirrorBaseUrl = mustBuildMirrorBaseURL()
	MirrorBaseUrl = mirrorBaseUrl.String()
)

func buildMirrorURL() (*url.URL, error) {
	base, err := buildMirrorBaseURL()
	if err != nil {
		return nil, err
	}
	if s, ok := os.LookupEnv("MIRROR_PATH"); ok {
		u, err := base.Parse(s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid MIRROR_PATH %s", s)
		}

		return u, nil
	}
	u, err := base.Parse(defaultMirrorPath)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid default mirror path %q", defaultMirrorPath)
	}

	return u, nil
}

func buildMirrorBaseURL() (*url.URL, error) {
	if s, ok := os.LookupEnv("MIRROR_BASE_URL"); ok {
		u, err := url.Parse(s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid MIRROR_BASE_URL %s", s)
		}
		if u.Path != "" && u.Path != "/" {
			return nil, errors.Errorf("MIRROR_BASE_URL must not include a path component: %s", u.Path)
		}

		return u, nil
	}
	if s, ok := os.LookupEnv("MIRROR_HOST"); ok {
		u, err := url.Parse("http://" + s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid MIRROR_HOST %s:", s)
		}
		if u.Path != "" && u.Path != "/" {
			return nil, errors.New("MIRROR_HOST must not include a path component")
		}

		return u, nil
	}
	mirror := fmt.Sprintf("http://install.%s.packet.net", FacilityCode)
	u, err := url.Parse(mirror)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid default mirror host: %s", mirror)
	}

	return u, nil
}

func mustBuildMirrorURL() *url.URL {
	u, err := buildMirrorURL()
	if err != nil {
		panic(err)
	}

	return u
}

func mustBuildMirrorBaseURL() *url.URL {
	u, err := buildMirrorBaseURL()
	if err != nil {
		panic(err)
	}

	return u
}
