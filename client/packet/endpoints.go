package packet

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
)

const mimeJSON = "application/json"

// PostHardwareComponent - POSTs a HardwareComponent to the API.
func (c *Reporter) PostHardwareComponent(ctx context.Context, hardwareID client.HardwareID, body io.Reader) (*client.ComponentsResponse, error) {
	var response client.ComponentsResponse

	if err := c.Post(ctx, "/hardware/"+hardwareID.String()+"/components", mimeJSON, body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
func (c *Reporter) PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	if err := c.Post(ctx, "/hardware/"+id+"/events", mimeJSON, body, &res); err != nil {
		return "", err
	}

	return res.ID, nil
}
func (c *Reporter) PostHardwarePhoneHome(ctx context.Context, id string) error {
	return c.Post(ctx, "/hardware/"+id+"/phone-home", "", nil, nil)
}
func (c *Reporter) PostHardwareFail(ctx context.Context, id string, body io.Reader) error {
	return c.Post(ctx, "/hardware/"+id+"/fail", mimeJSON, body, nil)
}
func (c *Reporter) PostHardwareProblem(ctx context.Context, id client.HardwareID, body io.Reader) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	if err := c.Post(ctx, "/hardware/"+id.String()+"/problems", mimeJSON, body, &res); err != nil {
		return "", err
	}

	return res.ID, nil
}

func (c *Reporter) PostInstancePhoneHome(ctx context.Context, id string) error {
	return c.Post(ctx, "/devices/"+id+"/phone-home", "", nil, nil)
}

func (c *Reporter) PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	if err := c.Post(ctx, "/devices/"+id+"/events", mimeJSON, body, &res); err != nil {
		return "", err
	}

	return res.ID, nil
}
func (c *Reporter) PostInstanceFail(ctx context.Context, id string, body io.Reader) error {
	return c.Post(ctx, "/devices/"+id+"/fail", mimeJSON, body, nil)
}
func (c *Reporter) PostInstancePassword(ctx context.Context, id, pass string) error {
	var req = struct {
		Password string `json:"password"`
	}{
		Password: pass,
	}

	b, err := json.Marshal(&req)
	if err != nil {
		return errors.Wrap(err, "marshalling instance password")
	}

	return c.Post(ctx, "/devices/"+id+"/password", mimeJSON, bytes.NewReader(b), nil)
}
func (c *Reporter) UpdateInstance(ctx context.Context, id string, body io.Reader) error {
	return c.Patch(ctx, "/devices/"+id, mimeJSON, body, nil)
}
