package harvester

import (
	"context"
	"os"
	"testing"

	l "github.com/packethost/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/installers"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
)

var (
	testLogger l.Logger
)

func TestMain(m *testing.M) {
	logger, _ := l.Init("github.com/tinkerbell/boots")
	job.Init(logger)
	installers.Init(logger)
	testLogger = logger
	os.Exit(m.Run())
}

func TestBootScript(t *testing.T) {
	assert := require.New(t)

	mockJob := job.NewMock(t, "test.slug", "test.facility")
	mockJob.SetOSSlug("harvester")
	mockJob.SetMAC("00:00:ba:dd:be:ef")
	s := ipxe.NewScript()
	s.Set("iface", "eth0")
	s.Or("shell")
	s.Set("tinkerbell", "http://127.0.0.1")
	s.Set("syslog_host", "127.0.0.1")
	s.Set("ipxe_cloud_config", "packet")

	generateBootScript(context.Background(), mockJob.Job(), s)
	assert.Contains(string(s.Bytes()), "v1.0.2", "expected to find default harvester version")
}
