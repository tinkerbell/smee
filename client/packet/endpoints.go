package packet

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/cacher"
	"github.com/tinkerbell/boots/metrics"
)

const mimeJSON = "application/json"

// GetDiscoveryFromEM is called when Cacher returns an empty response for the MAC
// address. It does a POST to the Packet API /staff/cacher/hardware-discovery
// endpoint.
// This was split out from DiscoverHardwareFromDHCP to make the control flow
// easier to understand.
func (c *Reporter) GetDiscoveryFromEM(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (client.Discoverer, error) {
	if giaddr == nil {
		return nil, errors.New("missing MAC address")
	}

	labels := prometheus.Labels{"from": "dhcp"}
	metrics.HardwareDiscovers.With(labels).Inc()
	metrics.DiscoversInProgress.With(labels).Inc()
	defer metrics.DiscoversInProgress.With(labels).Dec()
	discoverTimer := prometheus.NewTimer(metrics.DiscoverDuration.With(labels))
	defer discoverTimer.ObserveDuration()

	req := struct {
		MAC       string `json:"mac"`
		GIADDR    string `json:"giaddr,omitempty"`
		CIRCUITID string `json:"circuit_id,omitempty"`
	}{
		MAC:       mac.String(),
		GIADDR:    giaddr.String(),
		CIRCUITID: circuitID,
	}

	b, err := json.Marshal(&req)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling api discovery")
	}

	c.logger.With("giaddr", req.GIADDR, "mac", req.MAC).Debug("hardware discovery by mac")

	var res cacher.DiscoveryCacher
	if err := c.Post(ctx, "/staff/cacher/hardware-discovery", mimeJSON, bytes.NewReader(b), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

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
