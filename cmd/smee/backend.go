package main

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/dhcp/backend/file"
	"github.com/tinkerbell/dhcp/backend/kube"
	"github.com/tinkerbell/dhcp/handler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Kube struct {
	// ConfigFilePath is the path to a kubernetes config file (kubeconfig).
	ConfigFilePath string
	// APIURL is the Kubernetes API URL.
	APIURL string
	// Namespace is an override for the Namespace the kubernetes client will watch.
	// The default is the Namespace the pod is running in.
	Namespace string
	Enabled   bool
}
type File struct {
	// FilePath is the path to a JSON FilePath containing hardware data.
	FilePath string
	Enabled  bool
}

func (k *Kube) getClient() (*rest.Config, error) {
	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: k.ConfigFilePath,
		},
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: k.APIURL,
			},
			Context: clientcmdapi.Context{
				Namespace: k.Namespace,
			},
		},
	)

	config, err := ccfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (k *Kube) backend(ctx context.Context) (handler.BackendReader, error) {
	config, err := k.getClient()
	if err != nil {
		return nil, err
	}

	kb, err := kube.NewBackend(config)
	if err != nil {
		return nil, err
	}

	go func() {
		err = kb.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	return kb, nil
}

func (s *File) backend(ctx context.Context, logger logr.Logger) (handler.BackendReader, error) {
	f, err := file.NewWatcher(logger, s.FilePath)
	if err != nil {
		return nil, err
	}

	go f.Start(ctx)

	return f, nil
}

// discoverTrustedProxies will use the Kubernetes client to discover the CIDR Ranges for Pods in cluster.
func (k *Kube) discoverTrustedProxies(ctx context.Context, l logr.Logger, trustedProxies []string) []string {
	config, err := k.getClient()
	if err != nil {
		l.Error(err, "failed to get Kubernetes client config")
		return nil
	}
	c, err := corev1client.NewForConfig(config)
	if err != nil {
		l.Error(err, "failed to create Kubernetes client")
		return nil
	}

	return combinedCIDRs(ctx, l, c, trustedProxies)
}

// combinedCIDRs returns the CIDR Ranges for Pods in cluster. Not all Kubernetes distributions provide a way to discover the entire podCIDR.
// Some distributions just provide the podCIDRs assigned to each node. combinedCIDRs tries all known locations where pod CIDRs might exist.
// For example, if a cluster has 3 nodes, each with a /24 podCIDR, and the cluster has a /16 podCIDR, combinedCIDRs will return 4 CIDR ranges.
func combinedCIDRs(ctx context.Context, l logr.Logger, c *corev1client.CoreV1Client, trustedProxies []string) []string {
	if podCIDRS, err := perNodePodCIDRs(ctx, c); err == nil {
		trustedProxies = append(trustedProxies, podCIDRS...)
	} else {
		l.V(1).Info("failed to get per node podCIDRs", "err", err)
	}

	if clusterCIDR, err := clusterPodCIDR(ctx, c); err == nil {
		trustedProxies = append(trustedProxies, clusterCIDR...)
	} else {
		l.V(1).Info("failed to get cluster wide podCIDR", "err", err)
	}

	return trustedProxies
}

// perNodePodCIDRs returns the CIDR Range for Pods on each node. This is the per node podCIDR as compared to the total podCIDR.
// This will get the podCIDR from each node in the cluster, not the entire cluster podCIDR. If a cluster grows after this is run,
// the new nodes will not be included until this func is run again.
// This should be used in conjunction with ClusterPodCIDR to be as complete and cross distribution compatible as possible.
func perNodePodCIDRs(ctx context.Context, c corev1client.CoreV1Interface) ([]string, error) {
	ns, err := c.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var trustedProxies []string
	for _, n := range ns.Items {
		trustedProxies = append(trustedProxies, n.Spec.PodCIDRs...)
	}

	return trustedProxies, nil
}

// clusterPodCIDR returns the CIDR Range for Pods in cluster. This is the total podCIDR as compared to the per node podCIDR.
// Some Kubernetes distributions do not run a kube-controller-manager pod, so this func should be used in conjunction with PerNodePodCIDRs
// to be as complete and cross distribution compatible as possible.
func clusterPodCIDR(ctx context.Context, c corev1client.CoreV1Interface) ([]string, error) {
	// https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
	pods, err := c.Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "component=kube-controller-manager",
	})
	if err != nil {
		return nil, err
	}

	var trustedProxies []string
	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			for _, e := range c.Command {
				if strings.HasPrefix(e, "--cluster-cidr") {
					trustedProxies = append(trustedProxies, strings.Split(e, "=")[1])
				}
			}
		}
	}

	return trustedProxies, nil
}
