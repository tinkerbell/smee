package kubernetes

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sReporter struct {
	clientFunc    func() crclient.Client
	logger        log.Logger
	eventRecorder record.EventRecorder
}

const (
	HardwarePhoneHome = "HardwarePhoneHome"
	HardwareUpdated   = "HardwareUpdated"
	HardwareProblem   = "HardwareProblem"
	HardwareEvent     = "HardwareEvent"
	InstanceEvent     = "InstanceEvent"
	HardwareFailure   = "HardwareFailure"
	InstanceFailure   = "InstanceFailure"
	InstancePassword  = "InstancePassword"
	InstancePhoneHome = "InstancePhoneHome"
)

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

	return newReporterFromConfig(logger, config)

}

func newReporterFromConfig(logger log.Logger, config *rest.Config) (*k8sReporter, error) {
	cluster, err := NewCluster(config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	runtime.Must(v1alpha1.AddToScheme(cluster.GetScheme()))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: k8sClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(cluster.GetScheme(), corev1.EventSource{Component: "boots"})

	go cluster.Start(context.Background())

	return &k8sReporter{
		logger:        logger,
		clientFunc:    cluster.GetClient,
		eventRecorder: recorder,
	}, nil
}

func (r *k8sReporter) PostHardwareComponent(ctx context.Context, hardwareID client.HardwareID, body io.Reader) (*client.ComponentsResponse, error) {
	return nil, nil
}

func (r *k8sReporter) PostHardwareEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	// k8s model uses instanceID as hardwareID
	return r.PostInstanceEvent(ctx, id, body)
}

func (r *k8sReporter) PostHardwarePhoneHome(ctx context.Context, id string) error {
	// in case of k8s model, the hardwareID is same as instanceID
	return r.PostInstancePhoneHome(ctx, id)
}

func (r *k8sReporter) PostHardwareProblem(ctx context.Context, id client.HardwareID, body io.Reader) (string, error) {
	hwObj, err := r.findHardwareByID(ctx, id.String())
	if err != nil {
		return "", err
	}

	message, err := ioutil.ReadAll(body)
	if err != nil {
		return "", errors.Wrap(err, "error reading body")
	}

	r.eventRecorder.Event(hwObj, "Failure", HardwarePhoneHome, fmt.Sprintf("Hardware encountered problem: %s", string(message)))
	return "", nil

}

func (r *k8sReporter) PostInstancePhoneHome(ctx context.Context, id string) error {
	hwObj, err := r.findHardwareByID(ctx, id)
	if err != nil {
		return err
	}
	r.eventRecorder.Event(hwObj, "Normal", InstancePhoneHome, fmt.Sprintf("Hardware identified by id %s phoned home successfully", id))
	return nil
}

func (r *k8sReporter) PhoneInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	hwObj, err := r.findHardwareByID(ctx, id)
	if err != nil {
		return "", err
	}

	message, err := ioutil.ReadAll(body)
	if err != nil {
		return "", errors.Wrap(err, "error reading body")
	}

	r.eventRecorder.Event(hwObj, "Normal", InstanceEvent, fmt.Sprintf("Hardware event: %s", string(message)))
	return "", nil
}

func (r *k8sReporter) PostHardwareFail(ctx context.Context, id string, body io.Reader) error {
	return r.PostInstanceFail(ctx, id, body)
}

func (r *k8sReporter) PostInstanceEvent(ctx context.Context, id string, body io.Reader) (string, error) {
	return r.PostInstanceEvent(ctx, id, body)
}

func (r *k8sReporter) PostInstanceFail(ctx context.Context, id string, body io.Reader) error {
	hwObj, err := r.findHardwareByID(ctx, id)
	if err != nil {
		return err
	}

	message, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "error reading body")
	}

	r.eventRecorder.Event(hwObj, "Failure", InstanceFailure, fmt.Sprintf("Hardware failure: %s", string(message)))
	return nil
}

func (r *k8sReporter) PostInstancePassword(ctx context.Context, id, pass string) error {
	hwObj, err := r.findHardwareByID(ctx, id)
	if err != nil {
		return err
	}

	r.eventRecorder.Event(hwObj, "Normal", InstancePassword, fmt.Sprintf("Instance password: %s", pass))
	return nil
}

func (r *k8sReporter) UpdateInstance(ctx context.Context, instanceID string, body io.Reader) error {

	if instanceID == "" {
		return fmt.Errorf("instance ID is empty")
	}

	hardwareList := &v1alpha1.HardwareList{}

	err := r.clientFunc().List(ctx, hardwareList, &crclient.ListOptions{})

	var matchingHardware []v1alpha1.Hardware

	for _, v := range hardwareList.Items {
		if v.Spec.Metadata.Instance.ID == instanceID {
			matchingHardware = append(matchingHardware, v)
		}
	}

	if len(matchingHardware) == 0 {
		return errors.New("no hardware found")
	}

	if len(matchingHardware) > 1 {
		return errors.Errorf("got %d hardware for instance %s, expected only 1", len(matchingHardware), instanceID)
	}

	hwObj := matchingHardware[0]
	r.logger.With("hardware", hwObj.GetName(), hwObj.GetNamespace()).Info("disabling PXE boot")
	err = patchInstance(&hwObj, body)
	if err != nil {
		return err
	}

	err = r.clientFunc().Update(ctx, &hwObj)
	if err != nil {
		return err
	}
	r.eventRecorder.Event(&hwObj, corev1.EventTypeNormal, HardwareUpdated, "PXEBoot disabled")
	return nil
}

func (r *k8sReporter) Post(ctx context.Context, ref, mime string, body io.Reader, v interface{}) error {
	return nil
}

func patchInstance(hw *v1alpha1.Hardware, body io.Reader) error {
	content, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Errorf("error converting update request: %v", err)
	}

	if strings.Contains(string(content), `{"allow_pxe":false}`) {
		for i, _ := range hw.Spec.Interfaces {
			hw.Spec.Interfaces[i].Netboot.AllowPXE = &[]bool{false}[0]
		}
	}

	return nil
}

func (r *k8sReporter) findHardwareByID(ctx context.Context, id string) (*v1alpha1.Hardware, error) {
	hardwareList := &v1alpha1.HardwareList{}

	err := r.clientFunc().List(ctx, hardwareList, &crclient.MatchingFields{
		InstanceIDIndex: id,
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed listing hardware")
	}

	if len(hardwareList.Items) == 0 {
		return nil, errors.New("no hardware found")
	}

	if len(hardwareList.Items) > 1 {
		return nil, errors.Errorf("got %d hardware for id %s, expected only 1", len(hardwareList.Items), id)
	}

	hwObj := hardwareList.Items[0]
	return &hwObj, nil
}
