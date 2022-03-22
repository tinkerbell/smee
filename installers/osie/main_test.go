package osie

import (
	"os"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/job"
)

func TestMain(m *testing.M) {
	os.Setenv("PACKET_ENV", "test")
	os.Setenv("PACKET_VERSION", "0")
	os.Setenv("ROLLBAR_DISABLE", "1")
	os.Setenv("ROLLBAR_TOKEN", "1")
	os.Setenv("OSIE_VENDOR_SERVICES_URL", "https://localhost")

	logger, _ := l.Init("github.com/tinkerbell/boots")
	job.Init(logger)
	os.Exit(m.Run())
}
