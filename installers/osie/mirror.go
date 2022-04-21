package osie

import (
	"os"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/installers"
)

var (
	osieURL                            = conf.MirrorBaseURL + "/misc/osie"
	dockerRegistry                     string
	grpcAuthority, grpcCertURL         string
	registryUsername, registryPassword string
)

func buildWorkerParams() {
	dockerRegistry = os.Getenv("DOCKER_REGISTRY")
	grpcAuthority = getParam("TINKERBELL_GRPC_AUTHORITY")
	grpcCertURL = getParam("TINKERBELL_CERT_URL")
	registryUsername = os.Getenv("REGISTRY_USERNAME")
	registryPassword = os.Getenv("REGISTRY_PASSWORD")
}

func getParam(key string) string {
	value := os.Getenv(key)
	if value == "" {
		installers.Logger("osie").With("key", key).Fatal(errors.New("invalid key"))
	}

	return value
}
