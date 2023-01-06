package job

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	dhcp4 "github.com/packethost/dhcp4-go"
	l "github.com/packethost/pkg/log"
	assert "github.com/stretchr/testify/require"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/standalone"
	"github.com/tinkerbell/boots/conf"
)

func TestMain(m *testing.M) {
	logger, _ := l.Init("github.com/tinkerbell/boots")
	Init(logger)
	os.Exit(m.Run())
}

func TestSetPXEFilename(t *testing.T) {
	conf.PublicFQDN = "boots-testing.packet.net"

	setPXEFilenameTests := []struct {
		name       string
		hState     string
		id         string
		iState     string
		slug       string
		plan       string
		allowPXE   bool
		packet     bool
		arm        bool
		uefi       bool
		httpClient bool
		filename   string
	}{
		{
			name:   "just in_use",
			hState: "in_use",
		},
		{
			name:   "no instance state",
			hState: "in_use", id: "$instance_id", iState: "",
		},
		{
			name:   "instance not active",
			hState: "in_use", id: "$instance_id", iState: "not_active",
		},
		{
			name:   "instance active",
			hState: "in_use", id: "$instance_id", iState: "active",
		},
		{
			name:   "active not custom ipxe",
			hState: "in_use", id: "$instance_id", iState: "active", slug: "not_custom_ipxe",
		},
		{
			name:   "active custom ipxe",
			hState: "in_use", id: "$instance_id", iState: "active", slug: "custom_ipxe",
			filename: "undionly.kpxe",
		},
		{
			name:   "active custom ipxe with allow pxe",
			hState: "in_use", id: "$instance_id", iState: "active", allowPXE: true,
			filename: "undionly.kpxe",
		},
		{
			name: "arm",
			arm:  true, filename: "snp.efi",
		},
		{
			name: "x86 uefi",
			uefi: true, filename: "ipxe.efi",
		},
		{
			name: "x86 uefi http client",
			uefi: true, allowPXE: true, httpClient: true,
			filename: "http://" + conf.PublicFQDN + "/ipxe/ipxe.efi",
		},
		{
			name:     "all defaults",
			filename: "undionly.kpxe",
		},
		{
			name:   "packet iPXE",
			packet: true, filename: "nonexistent",
		},
		{
			name:   "packet iPXE PXE allowed",
			packet: true, id: "$instance_id", allowPXE: true, filename: "http://" + conf.PublicFQDN + "/auto.ipxe",
		},
	}

	for i, tt := range setPXEFilenameTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("%+v", tt)

			if tt.plan == "" {
				tt.plan = "0"
			}

			instance := &client.Instance{
				ID:       tt.id,
				State:    client.InstanceState(tt.iState),
				AllowPXE: tt.allowPXE,
				OS: &client.OperatingSystem{
					OsSlug: tt.slug,
				},
				OSV: &client.OperatingSystem{
					OsSlug: tt.slug,
				},
			}
			j := Job{
				Logger: joblog.With("index", i, "hState", tt.hState, "id", tt.id, "iState", tt.iState, "slug", tt.slug, "plan", tt.plan, "allowPXE", tt.allowPXE, "packet", tt.packet, "arm", tt.arm, "uefi", tt.uefi, "filename", tt.filename),
				hardware: &standalone.HardwareStandalone{
					ID: "$hardware_id",
					Metadata: client.Metadata{
						State: client.HardwareState(tt.hState),
						Facility: client.Facility{
							PlanSlug: "baremetal_" + tt.plan,
						},
						Instance: instance,
					},
				},
				instance:     instance,
				NextServer:   conf.PublicIPv4,
				IpxeBaseURL:  conf.PublicFQDN + "/ipxe",
				BootsBaseURL: conf.PublicFQDN,
			}
			rep := dhcp4.NewPacket(42)
			j.setPXEFilename(&rep, tt.packet, tt.arm, tt.uefi, tt.httpClient)
			filename := string(bytes.TrimRight(rep.File(), "\x00"))

			if tt.filename != filename {
				t.Fatalf("unexpected filename want:%q, got:%q", tt.filename, filename)
			}
		})
	}
}

func TestAllowPXE(t *testing.T) {
	for _, tt := range []struct {
		want     bool
		hw       bool
		instance bool
		iid      string
	}{
		{want: true, hw: true},
		{want: false, hw: false, instance: true},
		{want: true, hw: false, instance: true, iid: "id"},
		{want: false, hw: false, instance: false, iid: "id"},
	} {
		name := fmt.Sprintf("want=%t, hardware=%t, instance=%t, instance_id=%s", tt.want, tt.hw, tt.instance, tt.iid)
		t.Run(name, func(t *testing.T) {
			j := Job{
				hardware: &standalone.HardwareStandalone{
					ID: "$hardware_id",
					Metadata: client.Metadata{
						Instance: &client.Instance{
							AllowPXE: tt.hw,
						},
					},
					Network: client.Network{
						Interfaces: []client.NetworkInterface{
							{
								Netboot: client.Netboot{
									AllowPXE: tt.hw,
								},
							},
						},
					},
				},
				instance: &client.Instance{
					ID:       tt.iid,
					AllowPXE: tt.instance,
				},
			}
			got := j.AllowPXE()
			if got != tt.want {
				t.Fatalf("unexpected return, want: %t, got %t", tt.want, got)
			}
		})
	}
}

func TestIsSpecialOS(t *testing.T) {
	t.Run("nil instance", func(t *testing.T) {
		special := IsSpecialOS(nil)
		assert.Equal(t, false, special)
	})

	for name, want := range map[string]bool{
		"custom_ipxe": true,
		"custom":      true,
		"vmware_foo":  true,
		"flatcar_foo": false,
	} {
		t.Run("OS-"+name, func(t *testing.T) {
			instance := &client.Instance{
				OS: &client.OperatingSystem{
					Slug: name,
				},
				OSV: &client.OperatingSystem{},
			}
			got := IsSpecialOS(instance)
			assert.Equal(t, want, got)
		})
		t.Run("OSV-"+name, func(t *testing.T) {
			instance := &client.Instance{
				OS: &client.OperatingSystem{},
				OSV: &client.OperatingSystem{
					Slug: name,
				},
			}
			got := IsSpecialOS(instance)
			assert.Equal(t, want, got)
		})
	}
}
