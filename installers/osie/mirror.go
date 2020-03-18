package osie

import (
	"net/url"
	"os"

	"github.com/packethost/boots/env"
	"github.com/pkg/errors"
)

const (
	defaultOsiePath = "/misc/osie"
)

var (
	osieURL = mustBuildOsieURL().String()
)

func buildOsieURL() (*url.URL, error) {
	base, err := url.Parse(env.MirrorBaseUrl)
	if err != nil {
		return nil, errors.Wrap(err, "parsing MirrorBaseUrl")
	}
	if s, ok := os.LookupEnv("OSIE_PATH"); ok {
		u, err := base.Parse(s)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid OSIE_PATH: %s", s)
		}
		return u, nil
	}
	u, err := base.Parse(defaultOsiePath)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid default osie path: %s", defaultOsiePath)
	}
	return u, nil
}

func mustBuildOsieURL() *url.URL {
	u, err := buildOsieURL()
	if err != nil {
		panic(err)
	}
	return u
}
