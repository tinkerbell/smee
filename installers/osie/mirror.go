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
	osieURL                    = mustBuildOSIEURL().String()
	osiePathOverride           = getOSIEPathOverride()
	mirrorBaseURL              = conf.MirrorBaseUrl
	dockerRegistry             string
	grpcAuthority, grpcCertURL string
	registryUsername           string
	registryPassword           string
	registryCertUrl            string		
	registryCertRequired       string		
	tinkWorkerImage            string
	useAbsoluteImageURI  string
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

func getOSIEPathOverride() string {
	base, err := url.Parse(conf.MirrorBaseUrl)
	if err != nil {
		panic(errors.Wrap(err, "parsing MirrorBaseUrl"))
	}
	if s, ok := os.LookupEnv("OSIE_PATH_OVERRIDE"); ok {
		u, err := base.Parse(s)
		if err != nil {
			panic(errors.Wrapf(err, "invalid OSIE_PATH_OVERRIDE: %s", s))
		}

		return u.String() 
	}
	return ""
}

func mustBuildOSIEURL() *url.URL {
	u, err := buildOSIEURL()
	if err != nil {
		panic(err)
	}

	return u
}

func buildWorkerParams() {
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
	grpcAuthority = getParam("TINKERBELL_GRPC_AUTHORITY")
	grpcCertURL = getParam("TINKERBELL_CERT_URL")
	registryUsername = os.Getenv("REGISTRY_USERNAME")
	registryPassword = os.Getenv("REGISTRY_PASSWORD")
	registryCertUrl = os.Getenv("REGISTRY_CERT_URL")
	registryCertRequired = os.Getenv("REGISTRY_CERT_REQUIRED")
	tinkWorkerImage = os.Getenv("TINK_WORKER_IMAGE")
	useAbsoluteImageURI = os.Getenv("USE_ABSOLUTE_IMAGE_URI")
}

func getParam(key string) string {
	value := os.Getenv(key)
	if value == "" {
		installers.Logger("osie").With("key", key).Fatal(errors.New("invalid key"))
	}

	return value
}
