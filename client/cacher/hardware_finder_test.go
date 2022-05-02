package cacher

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/packethost/cacher/protos/cacher"
	"github.com/tinkerbell/boots/client"
	mockcacher "github.com/tinkerbell/boots/client/cacher/mock_cacher"
)

func TestByIP(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.IP
		resp    *cacher.Hardware
		respErr error
		want    *DiscoveryCacher
		wantErr error
	}{
		{
			name:    "query error",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    nil,
			respErr: errors.New("no hardware"),
			want:    nil,
			wantErr: errors.New("get hardware by ip from cacher: no hardware"),
		},
		{
			name:    "not found error",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    &cacher.Hardware{JSON: ""},
			respErr: nil,
			want:    nil,
			wantErr: client.ErrNotFound,
		},
		{
			name:    "json error",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    &cacher.Hardware{JSON: "{"},
			respErr: nil,
			want:    nil,
			wantErr: errors.New("unmarshal json for discovery: unexpected end of JSON input"),
		},
		{
			name:    "happy path",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    &cacher.Hardware{JSON: `{"id": "abc123"}`},
			respErr: nil,
			want: &DiscoveryCacher{
				HardwareCacher: &HardwareCacher{
					ID: "abc123",
				},
				mac: nil,
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			cc := mockcacher.NewMockCacherClient(mockCtrl)
			cc.EXPECT().ByIP(context.Background(), &cacher.GetRequest{IP: tc.arg.String()}).Times(1).Return(tc.resp, tc.respErr)

			cf := HardwareFinder{cc}
			d, err := cf.ByIP(context.Background(), tc.arg)

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
			if diff := cmp.Diff(d.(*DiscoveryCacher), tc.want, cmp.AllowUnexported(DiscoveryCacher{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestByMAC(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.HardwareAddr
		resp    *cacher.Hardware
		respErr error
		want    *DiscoveryCacher
		wantErr error
	}{
		{
			name: "query error",
			arg: func() net.HardwareAddr {
				mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")

				return mac
			}(),
			resp:    nil,
			respErr: errors.New("no hardware"),
			want:    nil,
			wantErr: errors.New("get hardware by mac from cacher: no hardware"),
		},
		{
			name: "not found error",
			arg: func() net.HardwareAddr {
				mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")

				return mac
			}(),
			resp:    &cacher.Hardware{JSON: ""},
			respErr: nil,
			want:    nil,
			wantErr: client.ErrNotFound,
		},
		{
			name: "json error",
			arg: func() net.HardwareAddr {
				mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")

				return mac
			}(),
			resp:    &cacher.Hardware{JSON: "{"},
			respErr: nil,
			want:    nil,
			wantErr: errors.New("unmarshal json for discovery: unexpected end of JSON input"),
		},
		{
			name: "happy path",
			arg: func() net.HardwareAddr {
				mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")

				return mac
			}(),
			resp:    &cacher.Hardware{JSON: `{"id": "abc123"}`},
			respErr: nil,
			want: &DiscoveryCacher{
				HardwareCacher: &HardwareCacher{
					ID: "abc123",
				},
				mac: nil,
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			cc := mockcacher.NewMockCacherClient(mockCtrl)
			cc.EXPECT().ByMAC(context.Background(), &cacher.GetRequest{MAC: tc.arg.String()}).Times(1).Return(tc.resp, tc.respErr)

			cf := HardwareFinder{cc}
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
			if diff := cmp.Diff(d.(*DiscoveryCacher), tc.want, cmp.AllowUnexported(DiscoveryCacher{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
