package job

import (
	"bytes"
	"testing"

	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/tinkerbell/boots/env"
	"github.com/tinkerbell/boots/packet"
)

func TestSetPXEFilename(t *testing.T) {
	env.PublicFQDN = "boots-testing.packet.net"

	var setPXEFilenameTests = []struct {
		hState   string
		id       string
		iState   string
		slug     string
		plan     string
		allowPXE bool
		packet   bool
		arm      bool
		uefi     bool
		filename string
	}{
		{hState: "in_use"},
		{hState: "in_use", id: "$instance_id", iState: ""},
		{hState: "in_use", id: "$instance_id", iState: "not_active"},
		{hState: "in_use", id: "$instance_id", iState: "active"},
		{hState: "in_use", id: "$instance_id", iState: "active", slug: "not_custom_ipxe"},
		{hState: "in_use", id: "$instance_id", iState: "active", slug: "custom_ipxe",
			filename: "undionly.kpxe"},
		{hState: "in_use", id: "$instance_id", iState: "active", allowPXE: true,
			filename: "undionly.kpxe"},

		{plan: "hua",
			filename: "snp-hua.efi"},
		{plan: "2a2",
			filename: "snp-hua.efi"},
		{arm: true,
			filename: "snp-nolacp.efi"},
		{uefi: true,
			filename: "ipxe.efi"},
		{
			filename: "undionly.kpxe"},
		{packet: true,
			filename: "/nonexistent"},
		{packet: true, id: "$instance_id", allowPXE: true,
			filename: "http://" + env.PublicFQDN + "/auto.ipxe"},
	}

	for i, tt := range setPXEFilenameTests {
		t.Logf("index=%d hStahe=%q id=%q iState=%q slug=%q plan=%q allowPXE=%v packet=%v arm=%v uefi=%v filename=%q",
			i, tt.hState, tt.id, tt.iState, tt.slug, tt.plan, tt.allowPXE, tt.packet, tt.arm, tt.uefi, tt.filename)

		if tt.plan == "" {
			tt.plan = "0"
		}
		j := Job{
			Logger: joblog.With("index", i, "hStahe", tt.hState, "id", tt.id, "iState", tt.iState, "slug", tt.slug, "plan", tt.plan, "allowPXE", tt.allowPXE, "packet", tt.packet, "arm", tt.arm, "uefi", tt.uefi, "filename", tt.filename),
			hardware: &packet.Hardware{
				ID:       "$hardware_id",
				State:    packet.HardwareState(tt.hState),
				PlanSlug: "baremetal_" + tt.plan,
			},
			instance: &packet.Instance{
				ID:       tt.id,
				State:    packet.InstanceState(tt.iState),
				AllowPXE: tt.allowPXE,
				OS: packet.OperatingSystem{
					OsSlug: tt.slug,
				},
			},
		}
		rep := dhcp4.NewPacket(42)
		j.setPXEFilename(&rep, tt.packet, tt.arm, tt.uefi)
		filename := string(bytes.TrimRight(rep.File(), "\x00"))

		if tt.filename != filename {
			t.Fatalf("unexpected filename want:%q, got:%q", tt.filename, filename)
		}
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
		t.Logf("want=%t, hardware=%t, instance=%t, instance_id=%s",
			tt.want, tt.hw, tt.instance, tt.iid)
		j := Job{
			hardware: &packet.Hardware{
				AllowPXE: tt.hw,
			},
			instance: &packet.Instance{
				ID:       tt.iid,
				AllowPXE: tt.instance,
			},
		}
		got := j.isPXEAllowed()
		if got != tt.want {
			t.Fatalf("unexpected return, want: %t, got %t", tt.want, got)
		}
	}
}
