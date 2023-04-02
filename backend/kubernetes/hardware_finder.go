package kubernetes

import (
	"context"
	"net"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/backend"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	"github.com/tinkerbell/tink/pkg/controllers"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Finder is a type that looks up hardware and workflows from Kubernetes.
type Finder struct {
	clientFunc   func() crclient.Client
	cacheStarter func(context.Context) error
	logger       logr.Logger
}

// NewFinder returns a HardwareFinder that discovers hardware from Kubernetes.
//
// Callers must instantiate the client-side cache by calling Start() before use.
func NewFinder(logger logr.Logger, k8sAPI, kubeconfig, kubeNamespace string) (*Finder, error) {
	// TODO(moadqassem): Maybe use the tinkerbell kubeclient instead of using this cluster client similar to hegel.
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

	cluster, err := NewCluster(ccfg)
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
func (f *Finder) ByIP(ctx context.Context, ip net.IP) (backend.Discoverer, error) {
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
func (f *Finder) ByMAC(ctx context.Context, mac net.HardwareAddr, _ net.IP, _ string) (backend.Discoverer, error) {
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
