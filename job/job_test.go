package job

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/tinkerbell/boots/httplog"
	"github.com/tinkerbell/boots/packet"
	"github.com/packethost/pkg/log"
)

func TestMain(m *testing.M) {
	os.Setenv("PACKET_ENV", "test")
	os.Setenv("PACKET_VERSION", "0")
	os.Setenv("ROLLBAR_DISABLE", "1")
	os.Setenv("ROLLBAR_TOKEN", "1")

	joblog, _ = log.Init("github.com/tinkerbell/boots")
	httplog.Init(joblog)
	os.Exit(m.Run())
}

func TestSetupNil(t *testing.T) {
	d := &packet.Discovery{Hardware: &packet.Hardware{}}
	j1 := &Job{}
	j2 := &Job{}

	j1.setup(d)
	j1.Logger = log.Logger{}
	j2.Logger = j1.Logger
	if !reflect.DeepEqual(j1, j2) {
		fmt.Println(pretty.Compare(j1, j2))
		t.Fatal("jobs do not match")
	}
}

func TestSetupDiscover(t *testing.T) {
	macIPMI := packet.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	d := &packet.Discovery{
		Hardware: &packet.Hardware{
			Name:     "TestSetupDiscover",
			Instance: nil,
			NetworkPorts: []packet.Port{
				packet.Port{
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

	wantMode := modeManagement
	if j.mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v\n", wantMode, j.mode)
	}

	netConfig := d.IPMI
	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v\n", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v\n", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v\n", netConfig.Gateway, j.dhcp.Gateway())
	}
	if d.Hardware.Name != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v\n", d.Hardware.Name, j.dhcp.Hostname())
	}
}

// The easy way to differentiate between discovered hardware and enrolled/not-active hardware is by existence of PlanSLug
func TestSetupManagement(t *testing.T) {
	macIPMI := packet.MACAddr([6]byte{0x00, 0xDE, 0xAD, 0xBE, 0xEF, 0x00})
	d := &packet.Discovery{
		Hardware: &packet.Hardware{
			Name:     "TestSetupManagement",
			Instance: &packet.Instance{},
			PlanSlug: "f1.fake.x86",
			NetworkPorts: []packet.Port{
				packet.Port{
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

	wantMode := modeManagement
	if j.mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v\n", wantMode, j.mode)
	}

	netConfig := d.IPMI
	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v\n", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v\n", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v\n", netConfig.Gateway, j.dhcp.Gateway())
	}
	if d.Hardware.Name != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v\n", d.Name, j.dhcp.Hostname())
	}
}

func TestSetupInstance(t *testing.T) {
	d, macs, _ := MakeHardwareWithInstance()

	j := &Job{mac: macs[1].HardwareAddr()}
	j.setup(d)

	wantMode := modeInstance
	if j.mode != wantMode {
		t.Fatalf("incorect mode, want: %v, got: %v\n", wantMode, j.mode)
	}

	netConfig := d.NetConfig(macs[1].HardwareAddr())
	if !netConfig.Address.Equal(j.dhcp.Address()) {
		t.Fatalf("incorrect Address, want: %v, got: %v\n", netConfig.Address, j.dhcp.Address())
	}
	if !netConfig.Netmask.Equal(j.dhcp.Netmask()) {
		t.Fatalf("incorrect Netmask, want: %v, got: %v\n", netConfig.Netmask, j.dhcp.Netmask())
	}
	if !netConfig.Gateway.Equal(j.dhcp.Gateway()) {
		t.Fatalf("incorrect Gateway, want: %v, got: %v\n", netConfig.Gateway, j.dhcp.Gateway())
	}
	if d.Instance.Hostname != j.dhcp.Hostname() {
		t.Fatalf("incorrect Hostname, want: %v, got: %v\n", d.Instance.Hostname, j.dhcp.Hostname())
	}
}

func TestSetupFails(t *testing.T) {
	d := &packet.Discovery{Hardware: &packet.Hardware{}}
	j := &Job{}

	err := j.setup(d)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	// should still be able to log, see #_incident-130
	j.With("happyThoughts", true).Error(err)
}
