package tinkerbell

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	mockhardware "github.com/tinkerbell/boots/client/tinkerbell/mock_hardware"
	mockworkflow "github.com/tinkerbell/boots/client/tinkerbell/mock_workflow"
	tinkhardware "github.com/tinkerbell/tink/protos/hardware"
	tinkworkflow "github.com/tinkerbell/tink/protos/workflow"
)

func TestByIP(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.IP
		resp    *tinkhardware.Hardware
		respErr error
		want    *DiscoveryTinkerbellV1
		wantErr error
	}{
		{
			name:    "query error",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    nil,
			respErr: errors.New("no hardware"),
			want:    nil,
			wantErr: errors.New("get hardware by ip from tink: no hardware"),
		},
		{
			name:    "not found error",
			arg:     net.ParseIP("192.168.1.1"),
			resp:    &tinkhardware.Hardware{},
			respErr: nil,
			want:    nil,
			wantErr: client.ErrNotFound,
		},
		{
			name: "json error",
			arg:  net.ParseIP("192.168.1.1"),
			resp: &tinkhardware.Hardware{
				Id:       "abc123",
				Metadata: `{"state": "ready",`,
			},
			respErr: nil,
			want:    nil,
			wantErr: errors.New("marshal json for discovery: json: error calling MarshalJSON for type *pkg.HardwareWrapper: unexpected end of JSON input"),
		},
		{
			name: "happy path",
			arg:  net.ParseIP("192.168.1.1"),
			// arg:     net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45},
			resp: &tinkhardware.Hardware{
				Id:       "abc123",
				Metadata: `{"state": "ready", "provisioner_engine": "tinkerbell"}`,
			},
			respErr: nil,
			want: &DiscoveryTinkerbellV1{
				HardwareTinkerbellV1: &HardwareTinkerbellV1{
					ID: "abc123",

					Metadata: client.Metadata{
						State:             "ready",
						ProvisionerEngine: "tinkerbell",
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tcli := mockhardware.NewMockHardwareServiceClient(mockCtrl)
			tcli.EXPECT().ByIP(context.Background(), &tinkhardware.GetRequest{Ip: tc.arg.String()}).Times(1).Return(tc.resp, tc.respErr)

			tinkfinder := HardwareFinder{tcli}
			d, err := tinkfinder.ByIP(context.Background(), tc.arg)
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
			if diff := cmp.Diff(d.(*DiscoveryTinkerbellV1), tc.want, cmp.AllowUnexported(DiscoveryTinkerbellV1{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestByMAC(t *testing.T) {
	cases := []struct {
		name    string
		arg     net.HardwareAddr
		resp    *tinkhardware.Hardware
		respErr error
		want    *DiscoveryTinkerbellV1
		wantErr error
	}{
		{
			name:    "query error",
			arg:     net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45},
			resp:    nil,
			respErr: errors.New("no hardware"),
			want:    nil,
			wantErr: errors.New("get hardware by mac from tink: no hardware"),
		},
		{
			name:    "not found error",
			arg:     net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45},
			resp:    &tinkhardware.Hardware{},
			respErr: nil,
			want:    nil,
			wantErr: client.ErrNotFound,
		},
		{
			name: "json error",
			arg:  net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45},
			resp: &tinkhardware.Hardware{
				Id:       "abc123",
				Metadata: `{"state": "ready",`,
			},
			respErr: nil,
			want:    nil,
			wantErr: errors.New("marshal json for discovery: json: error calling MarshalJSON for type *pkg.HardwareWrapper: unexpected end of JSON input"),
		},
		{
			name: "happy path",
			arg:  net.HardwareAddr{0xab, 0xcd, 0xef, 0x01, 0x23, 0x45},
			resp: &tinkhardware.Hardware{
				Id:       "abc123",
				Metadata: `{"state": "ready", "provisioner_engine": "tinkerbell"}`,
			},
			respErr: nil,
			want: &DiscoveryTinkerbellV1{
				HardwareTinkerbellV1: &HardwareTinkerbellV1{
					ID: "abc123",

					Metadata: client.Metadata{
						State:             "ready",
						ProvisionerEngine: "tinkerbell",
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tcli := mockhardware.NewMockHardwareServiceClient(mockCtrl)
			tcli.EXPECT().ByMAC(context.Background(), &tinkhardware.GetRequest{Mac: tc.arg.String()}).Times(1).Return(tc.resp, tc.respErr)

			tinkfinder := HardwareFinder{tcli}
			d, err := tinkfinder.ByMAC(context.Background(), tc.arg, net.ParseIP("1.1.1.1"), "")
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
			if diff := cmp.Diff(d.(*DiscoveryTinkerbellV1), tc.want, cmp.AllowUnexported(DiscoveryTinkerbellV1{})); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestWorkflowFinder(t *testing.T) {
	cases := []struct {
		name    string
		arg     client.HardwareID
		resp    *tinkworkflow.WorkflowContextList
		respErr error
		want    bool
		wantErr error
	}{
		{
			name:    "missing id error",
			arg:     client.HardwareID(""),
			resp:    nil,
			respErr: nil,
			want:    false,
			wantErr: errors.New("missing hardware id"),
		},
		{
			name:    "not found no error",
			arg:     client.HardwareID("hw1"),
			resp:    &tinkworkflow.WorkflowContextList{},
			respErr: nil,
			want:    false,
			wantErr: nil,
		},
		{
			name:    "fetching error",
			arg:     client.HardwareID("hw1"),
			resp:    nil,
			respErr: errors.New("something went wrong"),
			want:    false,
			wantErr: errors.New("error while fetching the workflow: something went wrong"),
		},
		{
			name: "happy path",
			arg:  client.HardwareID("hw1"),
			resp: &tinkworkflow.WorkflowContextList{
				WorkflowContexts: []*tinkworkflow.WorkflowContext{
					{
						CurrentActionState: tinkworkflow.State_STATE_PENDING,
					},
				},
			},
			respErr: nil,
			want:    true,
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tcli := mockworkflow.NewMockWorkflowServiceClient(mockCtrl)

			times := 1
			if tc.resp == nil && tc.respErr == nil {
				times = 0
			}
			tcli.EXPECT().GetWorkflowContextList(
				context.Background(),
				&tinkworkflow.WorkflowContextRequest{WorkerId: tc.arg.String()},
			).Times(times).Return(tc.resp, tc.respErr)

			tinkfinder := WorkflowFinder{tcli}
			got, err := tinkfinder.HasActiveWorkflow(context.Background(), tc.arg)
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
			if got != tc.want {
				t.Errorf("Got unexpected result: wanted %t, got %t", tc.want, got)
			}
		})
	}
}
