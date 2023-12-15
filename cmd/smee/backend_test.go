package main

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClusterPodCIDR(t *testing.T) {
	tests := map[string]struct {
		spec []runtime.Object
		want []string
	}{
		"no podCIDR": {},
		"podCIDR": {
			spec: []runtime.Object{
				&v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"component": "kube-controller-manager",
								},
								Namespace: "kube-system",
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Command: []string{
											"kube-controller-manager",
											"--allocate-node-cidrs=true",
											"--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf",
											"--authorization-kubeconfig=/etc/kubernetes/controller-manager.conf",
											"--bind-address=127.0.0.1",
											"--client-ca-file=/etc/kubernetes/pki/ca.crt",
											"--cluster-cidr=10.244.0.0/16",
											"--cluster-name=kubernetes",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []string{"10.244.0.0/16"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := fake.NewSimpleClientset(test.spec...)
			got, err := clusterPodCIDR(context.Background(), c.CoreV1())
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Fatalf("unexpected result (+want -got):\n%s", diff)
			}
		})
	}
}

func TestPerNodePodCIDRs(t *testing.T) {
	tests := map[string]struct {
		spec []runtime.Object
		want []string
	}{
		"no podCIDR": {},
		"podCIDR": {
			spec: []runtime.Object{
				&v1.NodeList{
					Items: []v1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.0.0/24"},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.1.0/24"},
							},
						},
					},
				},
			},
			want: []string{"10.42.0.0/24", "10.42.1.0/24"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := fake.NewSimpleClientset(test.spec...)
			got, err := perNodePodCIDRs(context.Background(), c.CoreV1())
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Fatalf("unexpected result (+want -got):\n%s", diff)
			}
		})
	}
}

func TestCombinedCIDRs(t *testing.T) {
	tests := map[string]struct {
		spec []runtime.Object
		want []string
	}{
		"no podCIDR": {},
		"podCIDR": {
			spec: []runtime.Object{
				&v1.NodeList{
					Items: []v1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.0.0/24"},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.1.0/24"},
							},
						},
					},
				},
				&v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"component": "kube-controller-manager",
								},
								Namespace: "kube-system",
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Command: []string{
											"kube-controller-manager",
											"--allocate-node-cidrs=true",
											"--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf",
											"--authorization-kubeconfig=/etc/kubernetes/controller-manager.conf",
											"--bind-address=127.0.0.1",
											"--client-ca-file=/etc/kubernetes/pki/ca.crt",
											"--cluster-cidr=10.244.0.0/16",
											"--cluster-name=kubernetes",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []string{"10.42.0.0/24", "10.42.1.0/24", "10.244.0.0/16"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := fake.NewSimpleClientset(test.spec...)
			got := combinedCIDRs(context.Background(), logr.Discard(), c.CoreV1(), nil)
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Fatalf("unexpected result (+want -got):\n%s", diff)
			}
		})
	}
}

func TestDiscoverTrustedProxies(t *testing.T) {
	t.Skip("dont think i can mock this")
	tests := map[string]struct {
		spec []runtime.Object
		want []string
	}{
		"no podCIDR": {},
		"podCIDR": {
			spec: []runtime.Object{
				&v1.NodeList{
					Items: []v1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.0.0/24"},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
							},
							Spec: v1.NodeSpec{
								PodCIDRs: []string{"10.42.1.0/24"},
							},
						},
					},
				},
				&v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"component": "kube-controller-manager",
								},
								Namespace: "kube-system",
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Command: []string{
											"kube-controller-manager",
											"--allocate-node-cidrs=true",
											"--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []string{"10.42.1.0/24", "10.42.0.0/24"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := &Kube{}
			got := k.discoverTrustedProxies(context.Background(), logr.Discard(), nil)
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Fatalf("unexpected result (+want -got):\n%s", diff)
			}
		})
	}
}
