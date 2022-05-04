package osie

import (
	"os"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/job"
)

func TestMain(m *testing.M) {
	logger, _ := l.Init("github.com/tinkerbell/boots")
	job.Init(logger)
	os.Exit(m.Run())
}
