package kube

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tinkerbell/smee/internal/dhcp/data"
	"github.com/tinkerbell/tink/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

func TestNewBackend(t *testing.T) {
	tests := map[string]struct {
		conf      *rest.Config
		opt       cluster.Option
		shouldErr bool
	}{
		"no config": {shouldErr: true},
		"failed index field": {shouldErr: true, conf: new(rest.Config), opt: func(o *cluster.Options) {
			cl := fake.NewClientBuilder().Build()
			o.NewClient = func(config *rest.Config, options client.Options) (client.Client, error) {
				return cl, nil
			}
			o.MapperProvider = func(c *rest.Config, httpClient *http.Client) (meta.RESTMapper, error) {
				return cl.RESTMapper(), nil
			}
		}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := NewBackend(tt.conf, tt.opt)
			if tt.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.shouldErr && err != nil {
				t.Fatal(err)
			}
			if !tt.shouldErr && b == nil {
				t.Fatal("expected backend")
			}
		})
	}
}

func TestToDHCPData(t *testing.T) {
	tests := map[string]struct {
		in        *v1alpha1.DHCP
		want      *data.DHCP
		shouldErr bool
	}{
		"nil input": {
			in:        nil,
			shouldErr: true,
		},
		"no mac": {
			in:        &v1alpha1.DHCP{},
			shouldErr: true,
		},
		"bad mac": {
			in:        &v1alpha1.DHCP{MAC: "bad"},
			shouldErr: true,
		},
		"no ip": {
			in:        &v1alpha1.DHCP{MAC: "aa:bb:cc:dd:ee:ff", IP: &v1alpha1.IP{}},
			shouldErr: true,
		},
		"no subnet": {
			in:        &v1alpha1.DHCP{MAC: "aa:bb:cc:dd:ee:ff", IP: &v1alpha1.IP{Address: "192.168.2.4"}},
			shouldErr: true,
		},
		"v1alpha1.IP == nil": {
			in:        &v1alpha1.DHCP{MAC: "aa:bb:cc:dd:ee:ff", IP: nil},
			shouldErr: true,
		},
		"bad gateway": {
			in:        &v1alpha1.DHCP{MAC: "aa:bb:cc:dd:ee:ff", IP: &v1alpha1.IP{Address: "192.168.2.4", Netmask: "255.255.254.0", Gateway: "bad"}},
			shouldErr: true,
		},
		"one bad nameserver": {
			in: &v1alpha1.DHCP{
				MAC:         "00:00:00:00:00:04",
				NameServers: []string{"1.1.1.1", "bad"},
				IP: &v1alpha1.IP{
					Address: "192.168.2.4",
					Netmask: "255.255.0.0",
					Gateway: "192.168.2.1",
				},
			},
			want: &data.DHCP{
				SubnetMask:     net.IPv4Mask(255, 255, 0, 0),
				DefaultGateway: netip.MustParseAddr("192.168.2.1"),
				NameServers:    []net.IP{net.IPv4(1, 1, 1, 1)},
				IPAddress:      netip.MustParseAddr("192.168.2.4"),
				MACAddress:     net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
			},
		},
		"full": {
			in: &v1alpha1.DHCP{
				MAC:         "00:00:00:00:00:04",
				Hostname:    "test",
				LeaseTime:   3600,
				NameServers: []string{"1.1.1.1"},
				IP: &v1alpha1.IP{
					Address: "192.168.1.4",
					Netmask: "255.255.255.0",
					Gateway: "192.168.1.1",
				},
			},
			want: &data.DHCP{
				SubnetMask:     net.IPv4Mask(255, 255, 255, 0),
				DefaultGateway: netip.MustParseAddr("192.168.1.1"),
				NameServers:    []net.IP{net.IPv4(1, 1, 1, 1)},
				Hostname:       "test",
				LeaseTime:      3600,
				IPAddress:      netip.MustParseAddr("192.168.1.4"),
				MACAddress:     net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := toDHCPData(tt.in)
			if tt.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(netip.Addr{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestToNetbootData(t *testing.T) {
	tests := map[string]struct {
		in        *v1alpha1.Netboot
		want      *data.Netboot
		shouldErr bool
	}{
		"nil input":    {in: nil, shouldErr: true},
		"bad ipxe url": {in: &v1alpha1.Netboot{IPXE: &v1alpha1.IPXE{URL: "bad"}}, shouldErr: true},
		"successful":   {in: &v1alpha1.Netboot{IPXE: &v1alpha1.IPXE{URL: "http://example.com/ipxe.ipxe"}}, want: &data.Netboot{IPXEScriptURL: &url.URL{Scheme: "http", Host: "example.com", Path: "/ipxe.ipxe"}}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := toNetbootData(tt.in, "")
			if tt.shouldErr && err == nil {
				t.Fatal("expected error")
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(netip.Addr{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestGetByIP(t *testing.T) {
	tests := map[string]struct {
		hwObject    []v1alpha1.Hardware
		wantDHCP    *data.DHCP
		wantNetboot *data.Netboot
		shouldErr   bool
		failToList  bool
	}{
		"empty hardware list":    {shouldErr: true, hwObject: []v1alpha1.Hardware{}},
		"more than one hardware": {shouldErr: true, hwObject: []v1alpha1.Hardware{hwObject1, hwObject2}},
		"bad dhcp data":          {shouldErr: true, hwObject: []v1alpha1.Hardware{badDHCPObject2}},
		"bad netboot data":       {shouldErr: true, hwObject: []v1alpha1.Hardware{badNetbootObject2}},
		"fail to list hardware":  {shouldErr: true, failToList: true},
		"good data": {hwObject: []v1alpha1.Hardware{hwObject1}, wantDHCP: &data.DHCP{
			MACAddress:     net.HardwareAddr{0x3c, 0xec, 0xef, 0x4c, 0x4f, 0x54},
			IPAddress:      netip.MustParseAddr("172.16.10.100"),
			SubnetMask:     []byte{0xff, 0xff, 0xff, 0x00},
			DefaultGateway: netip.MustParseAddr("255.255.255.0"),
			NameServers: []net.IP{
				{0x1, 0x1, 0x1, 0x1},
			},
			Hostname:  "sm01",
			LeaseTime: 86400,
			Arch:      "x86_64",
		}, wantNetboot: &data.Netboot{
			AllowNetboot: true,
			IPXEScriptURL: &url.URL{
				Scheme: "http",
				Host:   "netboot.xyz",
			},
			Facility: "onprem",
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rs := runtime.NewScheme()
			if err := scheme.AddToScheme(rs); err != nil {
				t.Fatal(err)
			}
			if err := v1alpha1.AddToScheme(rs); err != nil {
				t.Fatal(err)
			}

			ct := fake.NewClientBuilder()
			if !tc.failToList {
				ct = ct.WithScheme(rs)
				ct = ct.WithRuntimeObjects(&v1alpha1.HardwareList{})
				ct = ct.WithIndex(&v1alpha1.Hardware{}, IPAddrIndex, func(obj client.Object) []string {
					var list []string
					for _, elem := range tc.hwObject {
						list = append(list, elem.Spec.Interfaces[0].DHCP.IP.Address)
					}
					return list
				})
			}
			if len(tc.hwObject) > 0 {
				t.Logf("%+v", tc.hwObject[0].Spec.Interfaces[0].DHCP)
				t.Logf("%+v", tc.hwObject[0].Spec.Interfaces[0].DHCP.IP)
				ct = ct.WithLists(&v1alpha1.HardwareList{Items: tc.hwObject})
			}
			cl := ct.Build()

			fn := func(o *cluster.Options) {
				o.NewClient = func(config *rest.Config, options client.Options) (client.Client, error) {
					return cl, nil
				}
				o.MapperProvider = func(_ *rest.Config, _ *http.Client) (meta.RESTMapper, error) {
					return cl.RESTMapper(), nil
				}
				o.NewCache = func(config *rest.Config, options cache.Options) (cache.Cache, error) {
					return &informertest.FakeInformers{Scheme: cl.Scheme()}, nil
				}
			}
			rc := new(rest.Config)
			b, err := NewBackend(rc, fn)
			if err != nil {
				t.Fatal(err)
			}

			go b.Start(context.Background())
			gotDHCP, gotNetboot, err := b.GetByIP(context.Background(), net.IPv4(172, 16, 10, 100))
			if tc.shouldErr && err == nil {
				t.Log(err)
				t.Fatal("expected error")
			}

			if diff := cmp.Diff(gotDHCP, tc.wantDHCP, cmpopts.IgnoreUnexported(netip.Addr{})); diff != "" {
				t.Fatal(diff)
			}

			if diff := cmp.Diff(gotNetboot, tc.wantNetboot); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestGetByMac(t *testing.T) {
	tests := map[string]struct {
		hwObject    []v1alpha1.Hardware
		wantDHCP    *data.DHCP
		wantNetboot *data.Netboot
		shouldErr   bool
		failToList  bool
	}{
		"empty hardware list":    {shouldErr: true},
		"more than one hardware": {shouldErr: true, hwObject: []v1alpha1.Hardware{hwObject1, hwObject2}},
		"bad dhcp data":          {shouldErr: true, hwObject: []v1alpha1.Hardware{badDHCPObject}},
		"bad netboot data":       {shouldErr: true, hwObject: []v1alpha1.Hardware{badNetbootObject}},
		"fail to list hardware":  {shouldErr: true, failToList: true},
		"good data": {hwObject: []v1alpha1.Hardware{hwObject1}, wantDHCP: &data.DHCP{
			MACAddress:     net.HardwareAddr{0x3c, 0xec, 0xef, 0x4c, 0x4f, 0x54},
			IPAddress:      netip.MustParseAddr("172.16.10.100"),
			SubnetMask:     []byte{0xff, 0xff, 0xff, 0x00},
			DefaultGateway: netip.MustParseAddr("255.255.255.0"),
			NameServers: []net.IP{
				{0x1, 0x1, 0x1, 0x1},
			},
			Hostname:  "sm01",
			LeaseTime: 86400,
			Arch:      "x86_64",
		}, wantNetboot: &data.Netboot{
			AllowNetboot: true,
			IPXEScriptURL: &url.URL{
				Scheme: "http",
				Host:   "netboot.xyz",
			},
			Facility: "onprem",
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rs := runtime.NewScheme()
			if err := scheme.AddToScheme(rs); err != nil {
				t.Fatal(err)
			}
			if err := v1alpha1.AddToScheme(rs); err != nil {
				t.Fatal(err)
			}

			ct := fake.NewClientBuilder()
			if !tc.failToList {
				ct = ct.WithScheme(rs)
				ct = ct.WithRuntimeObjects(&v1alpha1.HardwareList{})
				ct = ct.WithIndex(&v1alpha1.Hardware{}, MACAddrIndex, func(obj client.Object) []string {
					var list []string
					for _, elem := range tc.hwObject {
						list = append(list, elem.Spec.Interfaces[0].DHCP.MAC)
					}
					return list
				})
			}
			if len(tc.hwObject) > 0 {
				t.Logf("%+v", tc.hwObject[0].Spec.Interfaces[0].DHCP)
				t.Logf("%+v", tc.hwObject[0].Spec.Interfaces[0].DHCP.MAC)
				ct = ct.WithLists(&v1alpha1.HardwareList{Items: tc.hwObject})
			}
			cl := ct.Build()

			fn := func(o *cluster.Options) {
				o.NewClient = func(config *rest.Config, options client.Options) (client.Client, error) {
					return cl, nil
				}
				o.MapperProvider = func(c *rest.Config, httpClient *http.Client) (meta.RESTMapper, error) {
					return cl.RESTMapper(), nil
				}
				o.NewCache = func(config *rest.Config, options cache.Options) (cache.Cache, error) {
					return &informertest.FakeInformers{Scheme: cl.Scheme()}, nil
				}
			}
			rc := new(rest.Config)
			b, err := NewBackend(rc, fn)
			if err != nil {
				t.Fatal(err)
			}

			go b.Start(context.Background())
			gotDHCP, gotNetboot, err := b.GetByMac(context.Background(), net.HardwareAddr{0x3c, 0xec, 0xef, 0x4c, 0x4f, 0x54})
			if tc.shouldErr && err == nil {
				t.Log(err)
				t.Fatal("expected error")
			}

			if diff := cmp.Diff(gotDHCP, tc.wantDHCP, cmpopts.IgnoreUnexported(netip.Addr{})); diff != "" {
				t.Fatal(diff)
			}

			if diff := cmp.Diff(gotNetboot, tc.wantNetboot); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

var hwObject1 = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine1",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Metadata: &v1alpha1.HardwareMetadata{
			Facility: &v1alpha1.MetadataFacility{
				FacilityCode: "onprem",
			},
		},
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					AllowPXE:      &[]bool{true}[0],
					AllowWorkflow: &[]bool{true}[0],
					IPXE: &v1alpha1.IPXE{
						URL: "http://netboot.xyz",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Arch:     "x86_64",
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.100",
						Gateway: "172.16.10.1",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:54",
					NameServers: []string{"1.1.1.1"},
					UEFI:        true,
				},
			},
		},
	},
}

var hwObject2 = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine2",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					AllowPXE:      &[]bool{true}[0],
					AllowWorkflow: &[]bool{true}[0],
					IPXE: &v1alpha1.IPXE{
						URL: "http://netboot.xyz",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Arch:     "x86_64",
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.101",
						Gateway: "172.16.10.1",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:55",
					NameServers: []string{"1.1.1.1"},
					UEFI:        true,
				},
			},
		},
		Metadata: &v1alpha1.HardwareMetadata{
			Facility: &v1alpha1.MetadataFacility{
				FacilityCode: "ewr2",
			},
		},
	},
}

var badDHCPObject = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine2",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					AllowPXE:      &[]bool{true}[0],
					AllowWorkflow: &[]bool{true}[0],
					IPXE: &v1alpha1.IPXE{
						URL: "http://netboot.xyz",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Arch:     "x86_64",
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.100",
						Gateway: "bad-address",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:54",
					NameServers: []string{"1.1.1.1"},
					UEFI:        true,
				},
			},
		},
	},
}

var badDHCPObject2 = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine2",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					AllowPXE:      &[]bool{true}[0],
					AllowWorkflow: &[]bool{true}[0],
					IPXE: &v1alpha1.IPXE{
						URL: "http://netboot.xyz",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Arch:     "x86_64",
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.100",
						Gateway: "bad-address",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:55",
					NameServers: []string{"1.1.1.1"},
					UEFI:        true,
				},
			},
		},
	},
}

var badNetbootObject = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine2",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					IPXE: &v1alpha1.IPXE{
						URL: "bad-url",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.101",
						Gateway: "172.16.10.1",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:54",
					NameServers: []string{"1.1.1.1"},
				},
			},
		},
	},
}

var badNetbootObject2 = v1alpha1.Hardware{
	TypeMeta: v1.TypeMeta{
		Kind:       "Hardware",
		APIVersion: "tinkerbell.org/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "machine2",
		Namespace: "default",
	},
	Spec: v1alpha1.HardwareSpec{
		Interfaces: []v1alpha1.Interface{
			{
				Netboot: &v1alpha1.Netboot{
					IPXE: &v1alpha1.IPXE{
						URL: "bad-url",
					},
				},
				DHCP: &v1alpha1.DHCP{
					Hostname: "sm01",
					IP: &v1alpha1.IP{
						Address: "172.16.10.100",
						Gateway: "172.16.10.1",
						Netmask: "255.255.255.0",
					},
					LeaseTime:   86400,
					MAC:         "3c:ec:ef:4c:4f:54",
					NameServers: []string{"1.1.1.1"},
				},
			},
		},
	},
}
