package client

import (
	"context"
	"io"

	"github.com/packethost/pkg/log"
)

type noOpReporter struct {
	logger log.Logger
}

func (c *noOpReporter) PostHardwareComponent(ctx context.Context, hardwareID HardwareID, body io.Reader) (*ComponentsResponse, error) {
	return nil, nil
}

func (c *noOpReporter) PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostHardwarePhoneHome(ctx context.Context, id string) error {
	return nil
}

func (c *noOpReporter) PostHardwareFail(ctx context.Context, id string, body io.Reader) error {
	return nil
}

func (c *noOpReporter) PostHardwareProblem(ctx context.Context, id HardwareID, body io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostInstancePhoneHome(context.Context, string) error {
	return nil
}

func (c *noOpReporter) PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostInstanceFail(ctx context.Context, id string, body io.Reader) error {
	return nil
}

func (c *noOpReporter) PostInstancePassword(ctx context.Context, id, pass string) error {
	return nil
}

func (c *noOpReporter) UpdateInstance(ctx context.Context, id string, body io.Reader) error {
	return nil
}

func (c *noOpReporter) Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	return nil
}

// NewNoOpReporter returns a reporter that does nothing. This is used for the
// Tinkerbell and standalone backends.
func NewNoOpReporter(logger log.Logger) Reporter {
	return &noOpReporter{
		logger: logger,
	}
}
