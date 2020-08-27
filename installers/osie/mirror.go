package osie

import (
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/installers"
)

const (
	defaultOSIEPath = "/misc/osie"
)

var (
	osieURL                                 = mustBuildOSIEURL().String()
	mirrorBaseURL                           = conf.MirrorBaseUrl
	dockerRegistry                          string
	grpcAuthority, grpcCertURL              string
	registryUsername, registryPassword      string
	log_driver, log_tag, log_server_address string
	log_server_address_type                 string
)

func buildOSIEURL() (*url.URL, error) {
	base, err := url.Parse(conf.MirrorBaseUrl)
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
	u, err := base.Parse(defaultOSIEPath)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid default osie path: %s", defaultOSIEPath)
	}
	return u, nil
}

func mustBuildOSIEURL() *url.URL {
	u, err := buildOSIEURL()
	if err != nil {
		panic(err)
	}
	return u
}

func buildWorkerParams() {
	dockerRegistry = getParam("DOCKER_REGISTRY")
	grpcAuthority = getParam("TINKERBELL_GRPC_AUTHORITY")
	grpcCertURL = getParam("TINKERBELL_CERT_URL")
	registryUsername = getParam("REGISTRY_USERNAME")
	registryPassword = getParam("REGISTRY_PASSWORD")
	log_driver = getParam("LOG_DRIVER")
	log_tag = getParam("LOG_TAG")
	log_server_address = getParam("LOG_SERVER_ADDRESS")
	log_server_address_type = getParam("LOG_SERVER_ADDRESS_TYPE")
}

func getParam(key string) string {
	value := os.Getenv(key)
	if value == "" {
		installers.Logger("osie").With("key", key).Fatal(errors.New("invalid key"))
	}
	return value
}
