package kubernetes

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	plog "github.com/packethost/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	cfg *rest.Config
	i   = &v1alpha1.Hardware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample",
			Namespace: "default",
		},
		Spec: v1alpha1.HardwareSpec{
			Interfaces: []v1alpha1.Interface{
				{
					DHCP: &v1alpha1.DHCP{
						MAC:      "3c:ec:ef:4c:4f:54",
						Arch:     "x86_64",
						Hostname: "sample",
						UEFI:     true,
					},
					Netboot: &v1alpha1.Netboot{
						AllowPXE:      &[]bool{true}[0],
						AllowWorkflow: &[]bool{true}[0],
					},
				},
				{
					DHCP: &v1alpha1.DHCP{
						MAC:      "3c:ec:ef:4c:4f:55",
						Arch:     "x86_64",
						Hostname: "sample",
						UEFI:     true,
					},
					Netboot: &v1alpha1.Netboot{
						AllowPXE:      &[]bool{true}[0],
						AllowWorkflow: &[]bool{true}[0],
					},
				},
			},
			Metadata: &v1alpha1.HardwareMetadata{
				Instance: &v1alpha1.MetadataInstance{
					ID:       "73009DBE-C6EB-4222-8DFB-79CFD6361B84",
					Userdata: "some data",
					AllowPxe: true,
				},
			},
		},
	}

	r *k8sReporter
)

func TestMain(t *testing.M) {

	testEnv := envtest.Environment{
		CRDDirectoryPaths: []string{
			"./testdata",
		},
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		log.Fatal(err)
	}

	l, err := plog.Init("k8sReporter-tests")
	if err != nil {
		log.Fatal(err)
	}
	r, err = newReporterFromConfig(l, cfg)
	if err != nil {
		log.Fatal(err)
	}

	runtime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	err = r.clientFunc().Create(context.TODO(), i)
	if err != nil {
		log.Fatal(err)
	}

	code := t.Run()
	testEnv.Stop()
	os.Exit(code)
}

func Test_UpdateInstance(t *testing.T) {
	assert := require.New(t)
	err := r.UpdateInstance(context.TODO(), "73009DBE-C6EB-4222-8DFB-79CFD6361B84", strings.NewReader(`{"allow_pxe":false}`))
	assert.NoError(err, "expected no error during update instance operation")
	obj := &v1alpha1.Hardware{}
	time.Sleep(10 * time.Second) // avoid flaky test
	err = r.clientFunc().Get(context.TODO(), types.NamespacedName{Namespace: i.Namespace, Name: i.Name}, obj)
	assert.NoError(err, "expected no error while fetching instance")
	for _, v := range obj.Spec.Interfaces {
		assert.Equal(false, *v.Netboot.AllowPXE, "expected PXE boot to be disabled")
	}
}
