package cacher

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/packethost/cacher/protos/cacher"
	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	mockcacher "github.com/tinkerbell/boots/client/cacher/mock_cacher"
	"github.com/tinkerbell/boots/metrics"
)

func TestMain(m *testing.M) {
	l, _ := log.Init("github.com/tinkerbell/boots")
	metrics.Init(l)
	os.Exit(m.Run())
}

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

			cf := HardwareFinder{cc, nil}
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
	giaddr := net.ParseIP("192.168.1.1")
	cases := []struct {
		name     string
		resp     *cacher.Hardware
		respErr  error
		want     *DiscoveryCacher
		wantErr  error
		reporter client.Reporter
	}{
		{
			name:    "query error",
			respErr: errors.New("no hardware"),
			wantErr: errors.New("get hardware by mac from cacher: no hardware"),
		},
		{
			name: "not found in cacher and not found in emapi",
			resp: &cacher.Hardware{JSON: ""},
			reporter: &fakeEMGetter{
				response: `{}`,
			},
			want: &DiscoveryCacher{},
		},
		{
			name: "not found in cacher found in emapi",
			resp: &cacher.Hardware{JSON: ""},
			want: &DiscoveryCacher{
				HardwareCacher: &HardwareCacher{
					ID: "abc123",
				},
			},
			reporter: &fakeEMGetter{
				response: `{"id":"abc123"}`,
			},
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

			cf := HardwareFinder{cc, tc.reporter}
			d, err := cf.ByMAC(context.Background(), mac, giaddr, "")
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

type fakeEMGetter struct {
	client.Reporter
	body     []byte
	response string
}

func (c *fakeEMGetter) Post(_ context.Context, _, _ string, body io.Reader, v interface{}) error {
	var err error
	c.body, err = io.ReadAll(body)
	if err != nil {
		panic(err)
	}

	return json.Unmarshal([]byte(c.response), v)
}
