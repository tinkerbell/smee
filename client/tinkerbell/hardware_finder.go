package tinkerbell

import (
	"context"
	"encoding/json"
	"net"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	tinkClient "github.com/tinkerbell/tink/client"
	tpkg "github.com/tinkerbell/tink/pkg"
	tinkhardware "github.com/tinkerbell/tink/protos/hardware"
	tinkworkflow "github.com/tinkerbell/tink/protos/workflow"
)

// HardwareFinder is a type that looks up hardware from Tinkerbell
type HardwareFinder struct {
	hClient tinkhardware.HardwareServiceClient
}

// NewHardwareFinder returns a Finder that discovers hardware from Tinkerbell.
//
// TODO: micahhausler: Explicitly pass in tink endpoint
func NewHardwareFinder() (*HardwareFinder, error) {
	hc, err := tinkClient.TinkHardwareClient()
	if err != nil {
		return nil, errors.Wrap(err, "connect to tink")
	}

	return &HardwareFinder{
		hClient: hc,
	}, nil
}

// ByIP returns a Discoverer for a particular IP.
func (f *HardwareFinder) ByIP(ctx context.Context, ip net.IP) (client.Discoverer, error) {
	resp, err := f.hClient.ByIP(ctx, &tinkhardware.GetRequest{
		Ip: ip.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by ip from tink")
	}
	// TODO: instead of marshaling/unmarshaling to JSON, just convert
	b, err := json.Marshal(&tpkg.HardwareWrapper{Hardware: resp}) // uses HardwareWrapper for its custom marshaler
	if err != nil {
		return nil, errors.Wrap(err, "marshal json for discovery")
	}
	if len(b) == 0 || string(b) == "{}" {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryTinkerbellV1{}
	err = json.Unmarshal(b, d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *HardwareFinder) ByMAC(ctx context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	resp, err := f.hClient.ByMAC(ctx, &tinkhardware.GetRequest{
		Mac: mac.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "get hardware by mac from tink")
	}
	// TODO: instead of marshaling/unmarshaling to JSON, just convert
	b, err := json.Marshal(&tpkg.HardwareWrapper{Hardware: resp}) // uses HardwareWrapper for its custom marshaler
	if err != nil {
		return nil, errors.Wrap(err, "marshal json for discovery")
	}
	if len(b) == 0 || string(b) == "{}" {
		return nil, client.ErrNotFound
	}
	d := &DiscoveryTinkerbellV1{}
	err = json.Unmarshal(b, d)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal json for discovery")
	}

	return d, nil
}

// NoOpWokrflowFinder is used to always return false.
type NoOpWorkflowFinder struct{}

// HasActiveWorkflow always returns false without error
func (f *NoOpWorkflowFinder) HasActiveWorkflow(context.Context, client.HardwareID) (bool, error) {
	return false, nil
}

// WorkflowFinder is a type for finding if a hardware ID has active workflows.
type WorkflowFinder struct {
	wClient tinkworkflow.WorkflowServiceClient
}

// NewFinder returns a *Finder that satisfies client.Finder.
//
// TODO: micahhausler: Explicitly pass in tink endpoint
func NewWorkflowFinder() (*WorkflowFinder, error) {
	wc, err := tinkClient.TinkWorkflowClient()
	if err != nil {
		return nil, errors.Wrap(err, "connect to tink")
	}

	return &WorkflowFinder{
		wClient: wc,
	}, nil
}

// HasActiveWorkflow finds if an active workflow exists for a particular hardware id.
func (f *WorkflowFinder) HasActiveWorkflow(ctx context.Context, hwID client.HardwareID) (bool, error) {
	if hwID == "" {
		return false, errors.New("missing hardware id")
	}

	// labels := prometheus.Labels{"from": "dhcp"}
	// cacherTimer := prometheus.NewTimer(metrics.CacherDuration.With(labels))
	// metrics.CacherRequestsInProgress.With(labels).Inc()
	// metrics.CacherTotal.With(labels).Inc()

	wcl, err := f.wClient.GetWorkflowContextList(ctx, &tinkworkflow.WorkflowContextRequest{WorkerId: hwID.String()})

	// cacherTimer.ObserveDuration()
	// metrics.CacherRequestsInProgress.With(labels).Dec()

	if err != nil {
		return false, errors.Wrap(err, "error while fetching the workflow")
	}

	for _, wf := range (*wcl).WorkflowContexts {
		if wf.CurrentActionState == tinkworkflow.State_STATE_PENDING || wf.CurrentActionState == tinkworkflow.State_STATE_RUNNING {
			return true, nil
		}
	}

	return false, nil
}
