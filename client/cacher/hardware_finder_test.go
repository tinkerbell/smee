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
	ip := net.ParseIP("192.168.1.1")
	cases := []struct {
		name    string
		resp    *cacher.Hardware
		respErr error
		want    *DiscoveryCacher
		wantErr error
	}{
		{
			name:    "query error",
			respErr: errors.New("no hardware"),
			wantErr: errors.New("get hardware by ip from cacher: no hardware"),
		},
		{
			name:    "not found error",
			resp:    &cacher.Hardware{JSON: ""},
			wantErr: client.ErrNotFound,
		},
		{
			name:    "json error",
			resp:    &cacher.Hardware{JSON: "{"},
			wantErr: errors.New("unmarshal json for discovery: unexpected end of JSON input"),
		},
		{
			name: "happy path",
			resp: &cacher.Hardware{JSON: `{"id": "abc123"}`},
			want: &DiscoveryCacher{
				HardwareCacher: &HardwareCacher{
					ID: "abc123",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			cc := mockcacher.NewMockCacherClient(mockCtrl)
			cc.EXPECT().ByIP(context.Background(), &cacher.GetRequest{IP: ip.String()}).Times(1).Return(tc.resp, tc.respErr)

			cf := HardwareFinder{cc}
			d, err := cf.ByIP(context.Background(), ip)

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
	mac, _ := net.ParseMAC("ab:cd:ef:01:12:34")
	cases := []struct {
		name    string
		resp    *cacher.Hardware
		respErr error
		want    *DiscoveryCacher
		wantErr error
	}{
		{
			name:    "query error",
			respErr: errors.New("no hardware"),
			wantErr: errors.New("get hardware by mac from cacher: no hardware"),
		},
		{
			name:    "not found error",
			resp:    &cacher.Hardware{JSON: ""},
			wantErr: client.ErrNotFound,
		},
		{
			name:    "json error",
			resp:    &cacher.Hardware{JSON: "{"},
			wantErr: errors.New("unmarshal json for discovery: unexpected end of JSON input"),
		},
		{
			name: "happy path",
			resp: &cacher.Hardware{JSON: `{"id": "abc123"}`},
			want: &DiscoveryCacher{
				HardwareCacher: &HardwareCacher{
					ID: "abc123",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			cc := mockcacher.NewMockCacherClient(mockCtrl)
			cc.EXPECT().ByMAC(context.Background(), &cacher.GetRequest{MAC: mac.String()}).Times(1).Return(tc.resp, tc.respErr)

			cf := HardwareFinder{cc}
			d, err := cf.ByMAC(context.Background(), mac, nil, "")

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
