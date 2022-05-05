package kubernetes

import (
	"context"
	"fmt"

	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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
func NewCluster(config *rest.Config) (cluster.Cluster, error) {
	runtimescheme := runtime.NewScheme()

	err := clientgoscheme.AddToScheme(runtimescheme)
	if err != nil {
		return nil, err
	}

	err = v1alpha1.AddToScheme(runtimescheme)
	if err != nil {
		return nil, err
	}

	c, err := cluster.New(config, func(o *cluster.Options) {
		o.Scheme = runtimescheme
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
			workflowWorkerNonTerminalStateIndexFunc,
		},
		{
			&v1alpha1.Hardware{},
			HardwareIPAddrIndex,
			hardwareIPIndexFunc,
		},
		{
			&v1alpha1.Hardware{},
			HardwareMACAddrIndex,
			hardwareMacIndexFunc,
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

// TODO micahhausler: make the following index functions public in tinkerbell/tink import from there

// workflowWorkerNonTerminalStateIndexFunc func indexes workflow by worker for non terminal workflows.
func workflowWorkerNonTerminalStateIndexFunc(obj client.Object) []string {
	wf, ok := obj.(*v1alpha1.Workflow)
	if !ok {
		return nil
	}

	resp := []string{}
	if !(wf.Status.State == v1alpha1.WorkflowStateRunning || wf.Status.State == v1alpha1.WorkflowStatePending) {
		return resp
	}
	for _, task := range wf.Status.Tasks {
		if task.WorkerAddr != "" {
			resp = append(resp, task.WorkerAddr)
		}
	}

	return resp
}

// hardwareMacIndexFunc returns a list of mac addresses from a hardware.
func hardwareMacIndexFunc(obj client.Object) []string {
	hw, ok := obj.(*v1alpha1.Hardware)
	if !ok {
		return nil
	}
	resp := []string{}
	for _, iface := range hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.MAC != "" {
			resp = append(resp, iface.DHCP.MAC)
		}
	}

	return resp
}

// hardwareIPIndexFunc returns a list of mac addresses from a hardware.
func hardwareIPIndexFunc(obj client.Object) []string {
	hw, ok := obj.(*v1alpha1.Hardware)
	if !ok {
		return nil
	}
	resp := []string{}
	for _, iface := range hw.Spec.Interfaces {
		if iface.DHCP != nil && iface.DHCP.IP != nil && iface.DHCP.IP.Address != "" {
			resp = append(resp, iface.DHCP.IP.Address)
		}
	}

	return resp
}
