package kubernetes

import (
	"context"
	"net"

	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	"github.com/tinkerbell/tink/pkg/controllers"
	"github.com/tinkerbell/tink/pkg/convert"
	"github.com/tinkerbell/tink/protos/workflow"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Finder is a type that looks up hardware and workflows from Kubernetes.
type Finder struct {
	clientFunc   func() crclient.Client
	cacheStarter func(context.Context) error
	logger       log.Logger
}

// NewFinder returns a HardwareFinder that discovers hardware from Kubernetes.
//
// Callers must instantiate the client-side cache by calling Start() before use.
func NewFinder(logger log.Logger, k8sAPI, kubeconfig, kubeNamespace string) (*Finder, error) {
	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeconfig,
		},
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: k8sAPI,
			},
			Context: clientcmdapi.Context{
				Namespace: kubeNamespace,
			},
		},
	)

	config, err := ccfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	cluster, err := NewCluster(config, kubeNamespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Finder{
		clientFunc:   cluster.GetClient,
		cacheStarter: cluster.Start,
		logger:       logger,
	}, nil
}

// Start instantiates the client-side cache.
func (f *Finder) Start(ctx context.Context) error {
	return f.cacheStarter(ctx)
}

// ByIP returns a Discoverer for a particular IP.
func (f *Finder) ByIP(ctx context.Context, ip net.IP) (client.Discoverer, error) {
	hardwareList := &v1alpha1.HardwareList{}

	err := f.clientFunc().List(ctx, hardwareList, &crclient.MatchingFields{
		controllers.HardwareIPAddrIndex: ip.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed listing hardware")
	}

	if len(hardwareList.Items) == 0 {
		return nil, errors.New("no hardware found")
	}

	if len(hardwareList.Items) > 1 {
		return nil, errors.Errorf("got %d hardware for ip %s, expected only 1", len(hardwareList.Items), ip)
	}

	return NewK8sDiscoverer(&hardwareList.Items[0]), nil
}

// ByMAC returns a Discoverer for a particular MAC address.
func (f *Finder) ByMAC(ctx context.Context, mac net.HardwareAddr, _ net.IP, _ string) (client.Discoverer, error) {
	hardwareList := &v1alpha1.HardwareList{}

	err := f.clientFunc().List(ctx, hardwareList, &crclient.MatchingFields{
		controllers.HardwareMACAddrIndex: mac.String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed listing hardware")
	}

	if len(hardwareList.Items) == 0 {
		return nil, errors.New("no hardware found")
	}

	if len(hardwareList.Items) > 1 {
		return nil, errors.Errorf("got %d hardware for mac %s, expected only 1", len(hardwareList.Items), mac)
	}

	return NewK8sDiscoverer(&hardwareList.Items[0]), nil
}

// HasActiveWorkflow finds if an active workflow exists for a particular hardware ID.
func (f *Finder) HasActiveWorkflow(ctx context.Context, hwID client.HardwareID) (bool, error) {
	if hwID == "" {
		return false, errors.New("missing hardware id")
	}

	stored := &v1alpha1.WorkflowList{}
	err := f.clientFunc().List(ctx, stored, &crclient.MatchingFields{
		controllers.WorkflowWorkerNonTerminalStateIndex: hwID.String(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list workflows")
	}

	wfContexts := []*workflow.WorkflowContext{}
	for _, wf := range stored.Items {
		wf := wf
		wfContexts = append(wfContexts, convert.WorkflowToWorkflowContext(&wf))
	}

	wcl := &workflow.WorkflowContextList{
		WorkflowContexts: wfContexts,
	}

	for _, wf := range wcl.WorkflowContexts {
		if wf.CurrentActionState == workflow.State_STATE_PENDING || wf.CurrentActionState == workflow.State_STATE_RUNNING {
			return true, nil
		}
	}

	return false, nil
}
