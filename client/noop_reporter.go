package client

import (
	"context"
	"io"

	"github.com/packethost/pkg/log"
)

type noOpReporter struct {
	logger log.Logger
}

func (c *noOpReporter) PostHardwareComponent(context.Context, HardwareID, io.Reader) (*ComponentsResponse, error) {
	return nil, nil
}

func (c *noOpReporter) PostHardwareEvent(context.Context, string, io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostHardwarePhoneHome(context.Context, string) error {
	return nil
}

func (c *noOpReporter) PostHardwareFail(context.Context, string, io.Reader) error {
	return nil
}

func (c *noOpReporter) PostHardwareProblem(context.Context, HardwareID, io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostInstancePhoneHome(context.Context, string) error {
	return nil
}

func (c *noOpReporter) PostInstanceEvent(context.Context, string, io.Reader) (string, error) {
	return "", nil
}

func (c *noOpReporter) PostInstanceFail(context.Context, string, io.Reader) error {
	return nil
}

func (c *noOpReporter) PostInstancePassword(context.Context, string, string) error {
	return nil
}

func (c *noOpReporter) UpdateInstance(context.Context, string, io.Reader) error {
	return nil
}

func (c *noOpReporter) Post(context.Context, string, string, io.Reader, interface{}) error {
	return nil
}

// NewNoOpReporter returns a reporter that does nothing. This is used for the
// Tinkerbell and standalone backends.
func NewNoOpReporter(logger log.Logger) Reporter {
	return &noOpReporter{
		logger: logger,
	}
}
