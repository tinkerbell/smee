package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/tinkerbell/dhcp/backend/file"
	"github.com/tinkerbell/dhcp/backend/kube"
	"github.com/tinkerbell/dhcp/handler"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type k8sConfig struct {
	// config is the path to a kubernetes config file.
	config string
	// api is the Kubernetes API URL.
	api string
	// namespace is an override for the namespace the kubernetes client will watch.
	namespace string
	enabled   bool
}

type standaloneConfig struct {
	// file is the path to a JSON file containing hardware data.
	file    string
	enabled bool
}

func getBackend(ctx context.Context, log logr.Logger, c *config) (handler.BackendReader, error) {
	switch c.backend {
	case "standalone":
		backend, err := c.standalone.standaloneBackend(ctx, log)
		if err != nil {
			return nil, err
		}

		return backend, nil
	case "kubernetes":
		backend, err := c.k8s.kubeBackend(ctx)
		if err != nil {
			return nil, err
		}

		return backend, nil
	default:
		return nil, fmt.Errorf("backend must be either standalone or kubernetes")
	}
}

func (k *k8sConfig) kubeBackend(ctx context.Context) (handler.BackendReader, error) {
	ccfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: k.config,
		},
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: k.api,
			},
			Context: clientcmdapi.Context{
				Namespace: k.namespace,
			},
		},
	)

	config, err := ccfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	kb, err := kube.NewBackend(config)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = kb.Start(ctx)
	}()

	return kb, nil
}

func (s *standaloneConfig) standaloneBackend(ctx context.Context, logger logr.Logger) (handler.BackendReader, error) {
	f, err := file.NewWatcher(logger, s.file)
	if err != nil {
		return nil, err
	}

	go f.Start(ctx)

	return f, nil
}

func (k *k8sConfig) validate(fs *flag.FlagSet) error {
	if k.enabled {
		if k.config == "" && k.api == "" {
			return fmt.Errorf("one of kubernetes config or api is required")
		}

		if k.namespace == "" {
			v := fs.Lookup("kubenamespace")
			fmt.Println("=========")
			fmt.Println(v)
			fmt.Println("=========")
			return fmt.Errorf("-%v ...\t%v is required", v.Name, v.Usage)
		}
	}

	return nil
}

func (s *standaloneConfig) validate(fs *flag.FlagSet) error {
	if s.enabled {
		if s.file == "" {
			v := fs.Lookup("standalonefile")
			return fmt.Errorf("%v is required", v.Usage)
		}
	}

	return nil
}
