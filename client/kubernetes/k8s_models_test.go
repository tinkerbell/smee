package kubernetes

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/tink/pkg/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstance(t *testing.T) {
	cases := []struct {
		name  string
		input *v1alpha1.Hardware
		want  *client.Instance
	}{
		{
			name: "nil metadata",
			input: &v1alpha1.Hardware{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1alpha1.HardwareSpec{
					Metadata: nil, // intentionally nil
				},
				Status: v1alpha1.HardwareStatus{},
			},
			want: nil,
		},
		{
			name: "nil metadata instance",
			input: &v1alpha1.Hardware{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1alpha1.HardwareSpec{
					Interfaces: []v1alpha1.Interface{},
					Metadata: &v1alpha1.HardwareMetadata{
						Instance: nil,
					},
				},
				Status: v1alpha1.HardwareStatus{},
			},
			want: nil,
		},
		{
			name: "real hardware",
			input: &v1alpha1.Hardware{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1alpha1.HardwareSpec{
					Interfaces: []v1alpha1.Interface{},
					Metadata: &v1alpha1.HardwareMetadata{
						State:       "",
						BondingMode: 0,
						Manufacturer: &v1alpha1.MetadataManufacturer{
							ID:   "",
							Slug: "",
						},
						Instance: &v1alpha1.MetadataInstance{
							ID:       "i-abcdef",
							State:    "active",
							Hostname: "ip-1-2-3-4.dns.local",
							AllowPxe: true,
							Rescue:   false,
							OperatingSystem: &v1alpha1.MetadataInstanceOperatingSystem{
								Distro:  "ubuntu",
								Version: "20.04",
								OsSlug:  "ubuntu_20_04",
							},
							AlwaysPxe:     false,
							IpxeScriptURL: "http://mumble.mumble.pxe/os",
							Ips: []*v1alpha1.MetadataInstanceIP{
								{
									Address:    "172.16.10.100",
									Netmask:    "255.255.255.0",
									Gateway:    "172.16.10.1",
									Family:     4,
									Public:     false,
									Management: false,
								},
							},
							Userdata:            "",
							CryptedRootPassword: "",
							Tags:                []string{},
							Storage:             &v1alpha1.MetadataInstanceStorage{},
							SSHKeys:             []string{},
							NetworkReady:        false,
						},
						Custom:   &v1alpha1.MetadataCustom{},
						Facility: &v1alpha1.MetadataFacility{},
					},
					TinkVersion: 0,
					Disks:       []v1alpha1.Disk{},
				},
				Status: v1alpha1.HardwareStatus{},
			},
			want: &client.Instance{
				ID:            "i-abcdef",
				State:         "active",
				Hostname:      "ip-1-2-3-4.dns.local",
				AllowPXE:      true,
				OS:            &client.OperatingSystem{Distro: "ubuntu", Version: "20.04", OsSlug: "ubuntu_20_04"},
				IPXEScriptURL: "http://mumble.mumble.pxe/os",
				IPs: []client.IP{
					{
						Address: net.ParseIP("172.16.10.100"),
						Netmask: net.ParseIP("255.255.255.0"),
						Gateway: net.ParseIP("172.16.10.1"),
						Family:  4,
					},
				},
				Tags:    []string{},
				SSHKeys: []string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewK8sDiscoverer(tc.input)
			got := d.Instance()
			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreUnexported(client.Instance{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
