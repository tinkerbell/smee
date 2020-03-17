package custom_ipxe

import (
	"os"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/packethost/tinkerbell/job"
)

func TestMain(m *testing.M) {
	os.Setenv("PACKET_ENV", "test")
	os.Setenv("PACKET_VERSION", "0")
	os.Setenv("ROLLBAR_DISABLE", "1")
	os.Setenv("ROLLBAR_TOKEN", "1")

	logger, _ := l.Init("github.com/packethost/tinkerbell")
	job.Init(logger)
	os.Exit(m.Run())
}
