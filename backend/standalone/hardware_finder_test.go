package standalone

import (
	"context"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/backend"
)

func TestByIP(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.IP
		db      []*DiscoverStandalone
		want    *DiscoverStandalone
		wantErr error
	}{
		{
			name: "not found",
			arg:  net.ParseIP("192.168.1.1"),
			db: []*DiscoverStandalone{
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc123",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										IP: backend.IP{
											Address: net.ParseIP("192.168.1.2"),
										},
									},
								},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: errors.New(`no hardware found for ip "192.168.1.1"`),
		},
		{
			name: "happy path",
			arg:  net.ParseIP("192.168.1.1"),
			db: []*DiscoverStandalone{
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc12",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										IP: backend.IP{
											Address: net.ParseIP("192.168.1.2"),
										},
									},
								},
							},
						},
					},
				},
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc123",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										IP: backend.IP{
											Address: net.ParseIP("192.168.1.1"),
										},
									},
								},
							},
						},
					},
				},
			},
			want: &DiscoverStandalone{
				HardwareStandalone: HardwareStandalone{
					ID: "abc123",
					Network: backend.Network{
						Interfaces: []backend.NetworkInterface{
							{
								DHCP: backend.DHCP{
									IP: backend.IP{
										Address: net.ParseIP("192.168.1.1"),
									},
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sf := HardwareFinder{
				db: tc.db,
			}
			d, err := sf.ByIP(context.Background(), tc.arg)
			if err != nil {
				if tc.wantErr == nil {
					t.Errorf("Unexpected error: %s", err)

					return
				}
				if tc.wantErr.Error() != err.Error() {
					t.Errorf("Unexpected error: got '%s', wanted '%s'", err, tc.wantErr)

					return
				}

				return
			}
			if err == nil && tc.wantErr != nil {
				t.Errorf("Missing expected error: got nil, wanted '%s'", tc.wantErr)

				return
			}
			if diff := cmp.Diff(d.(*DiscoverStandalone), tc.want, cmp.AllowUnexported(DiscoverStandalone{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestByMAC(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.HardwareAddr
		db      []*DiscoverStandalone
		want    *DiscoverStandalone
		wantErr error
	}{
		{
			name: "not found",
			arg: func() net.HardwareAddr {
				mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")

				return mac
			}(),
			db: []*DiscoverStandalone{
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc123",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										MAC: "00:00:00:00:00:00",
									},
								},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: errors.New(`no entry for MAC "ab:cd:ef:01:12:34" in standalone data`),
		},
		{
			name: "happy path",
			arg:  net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			db: []*DiscoverStandalone{
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc12",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										MAC: "00:00:00:00:00:00",
									},
								},
							},
						},
					},
				},
				{
					HardwareStandalone: HardwareStandalone{
						ID: "abc123",
						Network: backend.Network{
							Interfaces: []backend.NetworkInterface{
								{
									DHCP: backend.DHCP{
										MAC: "ff:ff:ff:ff:ff:ff",
									},
								},
							},
						},
					},
				},
			},
			want: &DiscoverStandalone{
				HardwareStandalone: HardwareStandalone{
					ID: "abc123",
					Network: backend.Network{
						Interfaces: []backend.NetworkInterface{
							{
								DHCP: backend.DHCP{
									MAC: "ff:ff:ff:ff:ff:ff",
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cf := HardwareFinder{tc.db}
			d, err := cf.ByMAC(context.Background(), tc.arg, nil, "")
			if err != nil {
				if tc.wantErr == nil {
					t.Errorf("Unexpected error: %s", err)

					return
				}
				if tc.wantErr.Error() != err.Error() {
					t.Errorf("Unexpected error: got '%s', wanted '%s'", err, tc.wantErr)

					return
				}

				return
			}
			if err == nil && tc.wantErr != nil {
				t.Errorf("Missing expected error: got nil, wanted '%s'", tc.wantErr)

				return
			}
			if diff := cmp.Diff(d.(*DiscoverStandalone), tc.want, cmp.AllowUnexported(DiscoverStandalone{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
