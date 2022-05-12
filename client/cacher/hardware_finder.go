package cacher

import (
	"bytes"
	"context"
	"encoding/json"
	"net"

	cacherClient "github.com/packethost/cacher/client"
	"github.com/packethost/cacher/protos/cacher"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/metrics"
)

// HardwareFinder is a type that can discover hardware from a cacher client.
type HardwareFinder struct {
	cc       cacher.CacherClient
	reporter client.Reporter
}

// NewHardwareFinder returns a github.com/packethost/cacher/client Finder.
func NewHardwareFinder(facility string, reporter client.Reporter) (*HardwareFinder, error) {
	cc, err := cacherClient.New(facility)
	if err != nil {
		return nil, errors.Wrap(err, "connect to cacher")
	}

	return &HardwareFinder{cc, reporter}, nil
}

// ByIP returns a Discoverer for a particular IP.
func (f *HardwareFinder) ByIP(ctx context.Context, ip net.IP) (client.Discoverer, error) {
	resp, err := f.cc.ByIP(ctx, &cacher.GetRequest{
		IP: ip.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by ip from cacher")
	}
	if len(resp.JSON) == 0 {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryCacher{}
	err = json.Unmarshal([]byte(resp.JSON), d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *HardwareFinder) ByMAC(ctx context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	resp, err := f.cc.ByMAC(ctx, &cacher.GetRequest{
		MAC: mac.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by mac from cacher")
	}
	if len(resp.JSON) == 0 {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryCacher{}
	err = json.Unmarshal([]byte(resp.JSON), d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}

// GetDiscoveryFromEM is called when Cacher returns an empty response for the MAC address.
// It does a POST to the Packet API /staff/cacher/hardware-discovery endpoint.
// This was split out from DiscoverHardwareFromDHCP to make the control flow easier to understand.
func GetDiscoveryFromEM(ctx context.Context, reporter client.Reporter, mac net.HardwareAddr, giaddr net.IP, circuitID string) (client.Discoverer, error) {
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

	res := &DiscoveryCacher{}
	if err := reporter.Post(ctx, "/staff/cacher/hardware-discovery", "application/json", bytes.NewReader(b), res); err != nil {
		return nil, err
	}

	return res, nil
}
