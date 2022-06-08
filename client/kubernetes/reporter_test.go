package kubernetes

import (
	"github.com/stretchr/testify/require"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestPatchInstance(t *testing.T) {

	i := &v1alpha1.Hardware{
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
			},
			Metadata: &v1alpha1.HardwareMetadata{
				Instance: &v1alpha1.MetadataInstance{
					ID:       "3c:ec:ef:4c:4f:54",
					Userdata: "some data",
					AllowPxe: true,
				},
			},
		},
	}

	err := patchInstance(i, strings.NewReader(`{"allow_pxe":false}`), "3c:ec:ef:4c:4f:54")
	assert := require.New(t)
	assert.NoError(err, "no error expected")
	assert.False(*i.Spec.Interfaces[0].Netboot.AllowPXE, "expected allow_pxe to be disabled")
	assert.Equal(i.Spec.Metadata.Instance.ID, "3c:ec:ef:4c:4f:54", "no changes expected")
}
