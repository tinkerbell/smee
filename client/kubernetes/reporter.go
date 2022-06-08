package kubernetes

import (
	"context"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	"github.com/tinkerbell/tink/pkg/controllers"
	"io"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type k8sReporter struct {
	clientFunc func() crclient.Client
}

func NewReporter(logger log.Logger, k8sAPI, kubeconfig, kubeNamespace string) (*k8sReporter, error) {
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

	cluster, err := NewCluster(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	go cluster.Start(context.Background())

	return &k8sReporter{
		clientFunc: cluster.GetClient,
	}, nil

}

func (r *k8sReporter) PostHardwareComponent(ctx context.Context, hardwareID client.HardwareID, body io.Reader) (*client.ComponentsResponse, error) {
	return nil, nil
}

func (r *k8sReporter) PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return "", nil
}

func (r *k8sReporter) PostHardwarePhoneHome(ctx context.Context, id string) error {
	// do nothing
	return nil
}

func (c *k8sReporter) PostHardwareProblem(ctx context.Context, id client.HardwareID, body io.Reader) (string, error) {
	return "", nil
}

func (c *k8sReporter) PostInstancePhoneHome(ctx context.Context, id string) error {
	return nil
}

func (c *k8sReporter) PhoneInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return "", nil
}

func (c *k8sReporter) PostHardwareFail(ctx context.Context, id string, body io.Reader) error {
	return nil
}

func (c *k8sReporter) PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return "", nil
}

func (c *k8sReporter) PostInstanceFail(ctx context.Context, id string, body io.Reader) error {
	return nil
}

func (c *k8sReporter) PostInstancePassword(ctx context.Context, id, pass string) error {
	return nil
}

func (c *k8sReporter) UpdateInstance(ctx context.Context, macString string, body io.Reader) error {

	hardwareList := &v1alpha1.HardwareList{}

	err := c.clientFunc().List(ctx, hardwareList, &crclient.MatchingFields{
		controllers.HardwareMACAddrIndex: macString,
	})

	if len(hardwareList.Items) == 0 {
		return errors.New("no hardware found")
	}

	if len(hardwareList.Items) > 1 {
		return errors.Errorf("got %d hardware for mac %s, expected only 1", len(hardwareList.Items), macString)
	}

	hwObj := hardwareList.Items[0]

	err = patchInstance(&hwObj, body, macString)
	if err != nil {
		return err
	}

	return c.clientFunc().Update(ctx, &hwObj)
}

func (c *k8sReporter) Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	return nil
}

func patchInstance(hw *v1alpha1.Hardware, body io.Reader, macString string) error {
	content, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Errorf("error converting update request: %v", err)
	}

	if strings.Contains(string(content), `{"allow_pxe":false}`) {
		for i, v := range hw.Spec.Interfaces {
			if v.DHCP.MAC == macString {
				hw.Spec.Interfaces[i].Netboot.AllowPXE = &[]bool{false}[0]
			}
		}
	}

	return nil
}
