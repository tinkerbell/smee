package job

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/cacher"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/metrics"
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
	macIPMI := client.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	var d client.Discoverer = &cacher.DiscoveryCacher{
		HardwareCacher: &cacher.HardwareCacher{
			Name:     "TestSetupDiscover",
			Instance: nil,
			NetworkPorts: []client.Port{
				{
					Type: "ipmi",
					Data: struct {
						MAC  *client.MACAddr `json:"mac"`
						Bond string          `json:"bond"`
					}{
						MAC: &macIPMI,
					},
				},
			},
			IPMI: client.IP{
				Address: net.ParseIP("192.168.0.2"),
				Gateway: net.ParseIP("192.168.0.1"),
				Netmask: net.ParseIP("192.168.0.255"),
			},
		},
	}
	l := log.Test(t, "test")
	j := &Job{
		mac:    macIPMI.HardwareAddr(),
		Logger: l,
	}
	j.setup(context.Background(), d)

	dh := d.Hardware()
	h := dh.(*cacher.HardwareCacher)

	mode := d.Mode()

	wantMode := "management"
	if mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v", wantMode, mode)
	}

	dc := d.(*cacher.DiscoveryCacher)
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
	macIPMI := client.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	var d client.Discoverer = &cacher.DiscoveryCacher{
		HardwareCacher: &cacher.HardwareCacher{
			Name:     "TestSetupManagement",
			Instance: &client.Instance{},
			PlanSlug: "f1.fake.x86",
			NetworkPorts: []client.Port{
				{
					Type: "ipmi",
					Data: struct {
						MAC  *client.MACAddr `json:"mac"`
						Bond string          `json:"bond"`
					}{
						MAC: &macIPMI,
					},
				},
			},
			IPMI: client.IP{
				Address: net.ParseIP("192.168.0.2"),
				Gateway: net.ParseIP("192.168.0.1"),
				Netmask: net.ParseIP("192.168.0.255"),
			},
		},
	}

	dh := d.Hardware()
	h := dh.(*cacher.HardwareCacher)
	l := log.Test(t, "test")
	j := &Job{
		mac:    macIPMI.HardwareAddr(),
		Logger: l,
	}
	j.setup(context.Background(), d)

	mode := d.Mode()

	wantMode := "management"
	if mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v", wantMode, mode)
	}

	dc := d.(*cacher.DiscoveryCacher)
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
	var d client.Discoverer
	var macs []client.MACAddr
	d, macs, _ = MakeHardwareWithInstance()
	l := log.Test(t, "test")
	j := &Job{
		mac:    macs[1].HardwareAddr(),
		Logger: l,
	}
	j.setup(context.Background(), d)

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
	var d client.Discoverer = &cacher.DiscoveryCacher{HardwareCacher: &cacher.HardwareCacher{}}
	j := &Job{Logger: log.Test(t, "test")}

	_, err := j.setup(context.Background(), d)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	// should still be able to log, see #_incident-130
	j.With("happyThoughts", true).Error(err)
}

func TestSetupWithoutInstance(t *testing.T) {
	d, mac := MakeHardwareWithoutInstance()
	j := &Job{mac: mac.HardwareAddr(), Logger: log.Test(t, "test")}
	j.setup(context.Background(), d)

	hostname, _ := d.Hostname()
	if hostname != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v", hostname, j.dhcp.Hostname())
	}
}
