package client

import (
	"context"
	"io"
)

type Reporter interface {
	PostHardwareComponent(ctx context.Context, hardwareID HardwareID, body io.Reader) (*ComponentsResponse, error)
	PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error)
	PostHardwarePhoneHome(ctx context.Context, id string) error
	PostHardwareFail(ctx context.Context, id string, body io.Reader) error
	PostHardwareProblem(ctx context.Context, id HardwareID, body io.Reader) (string, error)

	PostInstancePhoneHome(context.Context, string) error
	PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error)
	PostInstanceFail(ctx context.Context, id string, body io.Reader) error
	PostInstancePassword(ctx context.Context, id, pass string) error
	UpdateInstance(ctx context.Context, id string, body io.Reader) error

	Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error
}
