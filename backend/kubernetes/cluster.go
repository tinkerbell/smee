package kubernetes

import (
	"context"
	"fmt"

	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	"github.com/tinkerbell/tink/pkg/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	WorkflowWorkerNonTerminalStateIndex = ".status.state.nonTerminalWorker"
	HardwareMACAddrIndex                = ".spec.interfaces.dhcp.mac"
	HardwareIPAddrIndex                 = ".spec.interfaces.dhcp.ip"
)

// NewCluster returns a controller-runtime cluster.Cluster with the Tinkerbell runtime
// scheme registered, and indexers for:
// * Hardware by MAC address
// * Hardware by IP address
// * Workflows by worker address
//
// Callers must instantiate the client-side cache by calling Start() before use.
func NewCluster(config clientcmd.ClientConfig) (cluster.Cluster, error) {
	runtimescheme := runtime.NewScheme()

	err := clientgoscheme.AddToScheme(runtimescheme)
	if err != nil {
		return nil, err
	}

	err = v1alpha1.AddToScheme(runtimescheme)
	if err != nil {
		return nil, err
	}

	ns, _, err := config.Namespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get client namespace: %v", err)
	}

	cfg, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %v", err)
	}

	c, err := cluster.New(cfg, func(o *cluster.Options) {
		o.Scheme = runtimescheme
		o.Namespace = ns
	})
	if err != nil {
		return nil, err
	}
	indexers := []struct {
		obj          client.Object
		field        string
		extractValue client.IndexerFunc
	}{
		{
			&v1alpha1.Workflow{},
			WorkflowWorkerNonTerminalStateIndex,
			controllers.WorkflowWorkerNonTerminalStateIndexFunc,
		},
		{
			&v1alpha1.Hardware{},
			HardwareIPAddrIndex,
			controllers.HardwareIPIndexFunc,
		},
		{
			&v1alpha1.Hardware{},
			HardwareMACAddrIndex,
			controllers.HardwareMacIndexFunc,
		},
	}
	for _, indexer := range indexers {
		if err := c.GetFieldIndexer().IndexField(
			context.Background(),
			indexer.obj,
			indexer.field,
			indexer.extractValue,
		); err != nil {
			return nil, fmt.Errorf("failed to setup %s indexer, %w", indexer.field, err)
		}
	}

	return c, nil
}
