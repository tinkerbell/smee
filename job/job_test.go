package job

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/packethost/pkg/log"
	assert "github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/metrics"
	"github.com/tinkerbell/boots/packet"
	workflowMock "github.com/tinkerbell/boots/packet/mock_workflow"
	tw "github.com/tinkerbell/tink/protos/workflow"
)

func TestMain(m *testing.M) {
	os.Setenv("PACKET_ENV", "test")
	os.Setenv("PACKET_VERSION", "0")
	os.Setenv("ROLLBAR_DISABLE", "1")
	os.Setenv("ROLLBAR_TOKEN", "1")

	joblog, _ = log.Init("github.com/tinkerbell/boots")
	httplog.Init(joblog)
	metrics.Init(joblog)
	os.Exit(m.Run())
}

func TestSetupDiscover(t *testing.T) {
	macIPMI := packet.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	var d packet.Discovery = &packet.DiscoveryCacher{
		HardwareCacher: &packet.HardwareCacher{
			Name:     "TestSetupDiscover",
			Instance: nil,
			NetworkPorts: []packet.Port{
				{
					Type: "ipmi",
					Data: struct {
						MAC  *packet.MACAddr `json:"mac"`
						Bond string          `json:"bond"`
					}{
						MAC: &macIPMI,
					},
				},
			},
			IPMI: packet.IP{
				Address: net.ParseIP("192.168.0.2"),
				Gateway: net.ParseIP("192.168.0.1"),
				Netmask: net.ParseIP("192.168.0.255"),
			},
		},
	}

	j := &Job{mac: macIPMI.HardwareAddr()}
	j.setup(d)

	dh := d.Hardware()
	h := dh.(*packet.HardwareCacher)

	mode := d.Mode()

	wantMode := "management"
	if mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v", wantMode, mode)
	}

	dc := d.(*packet.DiscoveryCacher)
	netConfig := dc.HardwareIPMI()
	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v", netConfig.Gateway, j.dhcp.Gateway())
	}
	if h.Name != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v", h.Name, j.dhcp.Hostname())
	}
}

// The easy way to differentiate between discovered hardware and enrolled/not-active hardware is by existence of PlanSLug
func TestSetupManagement(t *testing.T) {
	macIPMI := packet.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	var d packet.Discovery = &packet.DiscoveryCacher{
		HardwareCacher: &packet.HardwareCacher{
			Name:     "TestSetupManagement",
			Instance: &packet.Instance{},
			PlanSlug: "f1.fake.x86",
			NetworkPorts: []packet.Port{
				{
					Type: "ipmi",
					Data: struct {
						MAC  *packet.MACAddr `json:"mac"`
						Bond string          `json:"bond"`
					}{
						MAC: &macIPMI,
					},
				},
			},
			IPMI: packet.IP{
				Address: net.ParseIP("192.168.0.2"),
				Gateway: net.ParseIP("192.168.0.1"),
				Netmask: net.ParseIP("192.168.0.255"),
			},
		},
	}

	dh := d.Hardware()
	h := dh.(*packet.HardwareCacher)

	j := &Job{mac: macIPMI.HardwareAddr()}
	j.setup(d)

	mode := d.Mode()

	wantMode := "management"
	if mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v", wantMode, mode)
	}

	dc := d.(*packet.DiscoveryCacher)
	netConfig := dc.HardwareIPMI()

	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v", netConfig.Gateway, j.dhcp.Gateway())
	}
	if h.Name != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v", h.Name, j.dhcp.Hostname())
	}
}

func TestSetupInstance(t *testing.T) {
	var d packet.Discovery
	var macs []packet.MACAddr
	d, macs, _ = MakeHardwareWithInstance()

	j := &Job{mac: macs[1].HardwareAddr()}
	j.setup(d)

	mode := d.Mode()

	wantMode := "instance"
	if mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v", wantMode, mode)
	}

	netConfig := d.GetIP(macs[1].HardwareAddr())
	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v", netConfig.Gateway, j.dhcp.Gateway())
	}
	if d.Instance().Hostname != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v", d.Instance().Hostname, j.dhcp.Hostname())
	}
}

func TestSetupFails(t *testing.T) {
	var d packet.Discovery = &packet.DiscoveryCacher{HardwareCacher: &packet.HardwareCacher{}}
	j := &Job{}

	err := j.setup(d)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	// should still be able to log, see #_incident-130
	j.With("happyThoughts", true).Error(err)
}

func TestHasActiveWorkflow(t *testing.T) {
	for _, test := range []struct {
		name   string
		wcl    *tw.WorkflowContextList
		status bool
	}{
		{name: "test pending workflow",
			wcl: &tw.WorkflowContextList{
				WorkflowContexts: []*tw.WorkflowContext{
					{
						WorkflowId:         "pending-fake-workflow-bde9-812726eff314",
						CurrentActionState: 0,
					},
				},
			},
			status: true,
		},
		{name: "test running workflow",
			wcl: &tw.WorkflowContextList{
				WorkflowContexts: []*tw.WorkflowContext{
					{
						WorkflowId:         "running-fake-workflow-bde9-812726eff314",
						CurrentActionState: 1,
					},
				},
			},
			status: true,
		},
		{name: "test inactive workflow",
			wcl: &tw.WorkflowContextList{
				WorkflowContexts: []*tw.WorkflowContext{
					{
						WorkflowId:         "inactive-fake-workflow-bde9-812726eff314",
						CurrentActionState: 4,
					},
				},
			},
			status: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ht := &httptest.Server{
				URL:         "FakeURL",
				Listener:    nil,
				EnableHTTP2: false,
				TLS:         &tls.Config{},
				Config:      &http.Server{},
			}
			u, err := url.Parse(ht.URL)
			if err != nil {
				t.Fatal(err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cMock := workflowMock.NewMockWorkflowServiceClient(ctrl)
			cMock.EXPECT().GetWorkflowContextList(gomock.Any(), gomock.Any()).Return(test.wcl, nil)
			c := packet.NewMockClient(u, cMock)
			SetClient(c)
			s, err := HasActiveWorkflow(context.Background(), "Hardware-fake-bde9-812726eff314")
			if err != nil {
				t.Fatal("error occured while testing")
			}
			assert.Equal(t, s, test.status)
		})
	}
}

func TestSetupWithoutInstance(t *testing.T) {
	d, mac := MakeHardwareWithoutInstance()
	j := &Job{mac: mac.HardwareAddr()}
	j.setup(d)

	hostname, _ := d.Hostname()
	if hostname != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v", hostname, j.dhcp.Hostname())
	}
}
