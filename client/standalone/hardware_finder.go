package standalone

import (
	"context"
	"encoding/json"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	tinkclient "github.com/tinkerbell/tink/client"
	tinkworkflow "github.com/tinkerbell/tink/protos/workflow"
)

// HardwareFinder is a type for statically looking up hardware.
type HardwareFinder struct {
	db []*DiscoverStandalone
}

// NewHardwareFinder returns a Finder given a JSON file that is formatted as a slice of
// DiscoverStandalone.
func NewHardwareFinder(path string) (*HardwareFinder, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %q", path)
	}
	db := []*DiscoverStandalone{}
	err = json.Unmarshal(content, &db)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse configuration file %q", path)
	}

	return &HardwareFinder{
		db: db,
	}, nil
}

// ByIP returns a Discoverer for a particular IP.
func (f *HardwareFinder) ByIP(_ context.Context, ip net.IP) (client.Discoverer, error) {
	for _, d := range f.db {
		for _, hip := range d.HardwareIPs() {
			if hip.Address.Equal(ip) {
				return d, nil
			}
		}
	}

	return nil, errors.Errorf("no hardware found for ip %q", ip)
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *HardwareFinder) ByMAC(_ context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	for _, d := range f.db {
		if d.MAC().String() == mac.String() {
			return d, nil
		}
	}

	return nil, errors.Errorf("no entry for MAC %q in standalone data", mac.String())
}

// WorkflowFinder is a type for finding if a hardware ID has active workflows.
type WorkflowFinder struct {
	wClient tinkworkflow.WorkflowServiceClient
}

// NewWorkflowFinder returns a *WorkflowFinder that satisfies client.WorkflowFinder.
//
// TODO: micahhausler: Explicitly pass in tink endpoint.
func NewWorkflowFinder() (*WorkflowFinder, error) {
	wc, err := tinkclient.TinkWorkflowClient()
	if err != nil {
		return nil, errors.Wrap(err, "connect to tink")
	}

	return &WorkflowFinder{
		wClient: wc,
	}, nil
}

// HasActiveWorkflow finds if an active workflow exists for a particular hardware ID.
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

	for _, wf := range wcl.WorkflowContexts {
		if wf.CurrentActionState == tinkworkflow.State_STATE_PENDING || wf.CurrentActionState == tinkworkflow.State_STATE_RUNNING {
			return true, nil
		}
	}

	return false, nil
}
