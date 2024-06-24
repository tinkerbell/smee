package main

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/smee/internal/backend/file"
	"github.com/tinkerbell/smee/internal/backend/kube"
	"github.com/tinkerbell/smee/internal/dhcp/handler"
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

type Noop struct {
	Enabled bool
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
