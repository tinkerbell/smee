package job

import (
	"context"
	"fmt"
	"strings"

	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/packet"
	"go.opentelemetry.io/otel/trace"
)

func IsSpecialOS(i *packet.Instance) bool {
	if i == nil {
		return false
	}
	var slug string
	if i.OSV.Slug != "" {
		slug = i.OSV.Slug
	}
	if i.OS.Slug != "" {
		slug = i.OS.Slug
	}

	return slug == "custom_ipxe" || slug == "custom" || strings.HasPrefix(slug, "vmware") || strings.HasPrefix(slug, "nixos")
}

// ServeDHCP responds to DHCP packets. Returns true if it replied. Returns false
// if it did not reply, often for good reason. If it was an error, error will be
// set.
func (j Job) ServeDHCP(ctx context.Context, w dhcp4.ReplyWriter, req *dhcp4.Packet) (bool, error) {
	span := trace.SpanFromContext(ctx)

	// If we are not the chosen provisioner for this piece of hardware
	// do not respond to the DHCP request
	if !j.areWeProvisioner() {
		return false, nil
	}

	// setup reply
	span.AddEvent("dhcp.NewReply")
	// only DISCOVER and REQUEST get replies; reply is nil for ignored reqs
	reply := dhcp.NewReply(w, req)
	if reply == nil {
		return false, nil // ignore the request
	}

	// configure DHCP
	if !j.configureDHCP(ctx, reply.Packet(), req) {
		return false, errors.New("unable to configure DHCP for yiaddr and DHCP options")
	}

	// send the DHCP response
	span.AddEvent("reply.Send()")
	if err := reply.Send(); err != nil {
		return false, err
	}

	return true, nil
}

func (j Job) configureDHCP(ctx context.Context, rep, req *dhcp4.Packet) bool {
	span := trace.SpanFromContext(ctx)
	if !j.dhcp.ApplyTo(rep) {
		return false
	}

	if dhcp.SetupPXE(ctx, rep, req) {
		isARM := dhcp.IsARM(req)
		if dhcp.Arch(req) != j.Arch() {
			span.AddEvent(fmt.Sprintf("arch mismatch: got %q and expected %q", dhcp.Arch(req), j.Arch()))
			j.With("dhcp", dhcp.Arch(req), "job", j.Arch()).Info("arch mismatch, using dhcp")
		}

		isUEFI := dhcp.IsUEFI(req)
		if isUEFI != j.IsUEFI() {
			j.With("dhcp", isUEFI, "job", j.IsUEFI()).Info("uefi mismatch, using dhcp")
		}

		isPacket := ipxe.IsPacketIPXE(req)
		if isPacket {
			ipxe.Setup(rep)
		}

		j.setPXEFilename(rep, isPacket, isARM, isUEFI)
	} else {
		span.AddEvent("did not SetupPXE because packet is not a PXE request")
	}

	return true
}

func (j Job) isPXEAllowed() bool {
	if j.hardware.HardwareAllowPXE(j.mac) {
		return true
	}
	if j.InstanceID() == "" {
		return false
	}

	return j.instance.AllowPXE
}

func (j Job) areWeProvisioner() bool {
	if j.hardware.HardwareProvisioner() == "" {
		return true
	}

	return j.hardware.HardwareProvisioner() == j.ProvisionerEngineName()
}

func (j Job) setPXEFilename(rep *dhcp4.Packet, isPacket, isARM, isUEFI bool) {
	if j.HardwareState() == "in_use" {
		if j.InstanceID() == "" {
			j.Error(errors.New("setPXEFilename called on a job with no instance"))

			return
		}

		if j.instance.State != "active" {
			j.With("hardware.state", j.HardwareState(), "instance.state", j.instance.State).Info("device should NOT be trying to PXE boot")

			return
		}

		// ignore custom_ipxe because we always do dhcp for it and we'll want to do /nonexistent filename so
		// nics don't timeout.... but why though?
		if !j.isPXEAllowed() && j.hardware.OperatingSystem().OsSlug != "custom_ipxe" {
			err := errors.New("device should NOT be trying to PXE boot")
			j.With("hardware.state", j.HardwareState(), "allow_pxe", j.isPXEAllowed(), "os", j.hardware.OperatingSystem().OsSlug).Info(err)

			return
		}
		// custom_ipxe or rescue
	}

	var filename string
	var pxeClient bool
	if !isPacket {
		if j.PArch() == "hua" || j.PArch() == "2a2" {
			filename = "snp-hua.efi"
		} else if isARM {
			filename = "snp-nolacp.efi"
		} else if isUEFI {
			filename = "ipxe.efi"
		} else {
			filename = "undionly.kpxe"
		}
	} else if !j.isPXEAllowed() {
		// Always honor allow_pxe.
		// We set a filename because if a machine is actually trying to PXE and nothing is sent it may hang for
		// a while waiting for any possible ProxyDHCP packets and it would delay booting to disks and phoning-home.
		//
		// Why we wait until here instead of sending the file name early on? I don't know. We should not need to
		// send our iPXE, boot into it, and then send /nonexistent afaik.
		//
		// TODO(mmlb) try to move this logic to much earlier in the function, maybe all the way as the first thing even.

		os := j.OperatingSystem()
		j.With("instance.state", j.instance.State, "os_slug", os.Slug, "os_distro", os.Distro, "os_version", os.Version).Info()
		pxeClient = true
		filename = "/nonexistent"
	} else {
		pxeClient = true
		filename = "http://" + conf.PublicFQDN + "/auto.ipxe"
	}

	if filename == "" {
		err := errors.New("no filename is set")
		j.Error(err)

		return
	}

	dhcp.SetFilename(rep, filename, conf.PublicIPv4, pxeClient)
}
