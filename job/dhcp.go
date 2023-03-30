package job

import (
	"context"
	"fmt"

	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/dhcp"
	"go.opentelemetry.io/otel/trace"
)

// ServeDHCP responds to DHCP packets. Returns true if it replied. Returns false
// if it did not reply, often for good reason. If it was an error, error will be
// set.
func (j *Job) ServeDHCP(ctx context.Context, w dhcp4.ReplyWriter, req *dhcp4.Packet) (bool, error) {
	span := trace.SpanFromContext(ctx)

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

func (j *Job) configureDHCP(ctx context.Context, rep, req *dhcp4.Packet) bool {
	span := trace.SpanFromContext(ctx)
	if !j.dhcp.ApplyTo(rep) {
		return false
	}

	if dhcp.SetupPXE(ctx, j.Logger, rep, req) {
		isARM := dhcp.IsARM(req)
		if dhcp.Arch(req) != j.Arch() {
			span.AddEvent(fmt.Sprintf("arch mismatch: got %q and expected %q", dhcp.Arch(req), j.Arch()))
			j.Logger.Info("arch mismatch, using dhcp", "dhcp", dhcp.Arch(req), "job", j.Arch())
		}

		isUEFI := dhcp.IsUEFI(req)
		if isUEFI != j.IsUEFI() {
			j.Logger.Info("uefi mismatch, using dhcp", "dhcp", isUEFI, "job", j.IsUEFI())
		}

		isTinkerbellIPXE := dhcp.IsTinkerbellIPXE(req)
		if isTinkerbellIPXE {
			dhcp.Setup(rep, j.PublicSyslogIPv4)
		}

		j.setPXEFilename(rep, isTinkerbellIPXE, isARM, isUEFI, dhcp.IsHTTPClient(req))
	} else {
		span.AddEvent("did not SetupPXE because packet is not a PXE request")
	}

	return true
}

func (j *Job) setPXEFilename(rep *dhcp4.Packet, isTinkerbellIPXE, isARM, isUEFI, isHTTPClient bool) {
	if j.HardwareState() == "in_use" {
		if j.InstanceID() == "" {
			j.Logger.Error(errors.New("setPXEFilename called on a job with no instance"), "setPXEFilename called on a job with no instance")

			return
		}

		if j.instance.State != "active" {
			j.Logger.Info("device should NOT be trying to PXE boot", "hardware.state", j.HardwareState(), "instance.state", j.instance.State)

			return
		}

		// ignore custom_ipxe because we always do dhcp for it and we'll want to do /nonexistent filename so
		// nics don't timeout.... but why though?
		if !j.AllowPXE() && j.hardware.OperatingSystem().OsSlug != "custom_ipxe" {
			j.Logger.Info("device should NOT be trying to PXE boot", "hardware.state", j.HardwareState(), "allow_pxe", j.AllowPXE(), "os", j.hardware.OperatingSystem().OsSlug)

			return
		}
		// custom_ipxe or rescue
	}

	var filename string
	httpPrefix := j.BootsBaseURL
	switch {
	case !isTinkerbellIPXE:
		httpPrefix = j.IpxeBaseURL
		switch {
		case isARM:
			filename = "snp.efi"
		case isUEFI:
			filename = "ipxe.efi"
		default:
			filename = "undionly.kpxe"
		}
	case !j.AllowPXE():
		// Always honor allow_pxe.
		// We set a filename because if a machine is actually trying to PXE and nothing is sent it may hang for
		// a while waiting for any possible ProxyDHCP packets and it would delay booting to disks and phoning-home.
		//
		// Why we wait until here instead of sending the file name early on? I don't know. We should not need to
		// send our iPXE, boot into it, and then send /nonexistent afaik.
		//
		// TODO(mmlb) try to move this logic to much earlier in the function, maybe all the way as the first thing even.

		os := j.OperatingSystem()
		j.Logger.Info("info", "instance.state", j.instance.State, "os_slug", os.Slug, "os_distro", os.Distro, "os_version", os.Version)
		filename = "nonexistent"
	default:
		isHTTPClient = true
		filename = "auto.ipxe"
	}

	if filename == "" {
		err := errors.New("no filename is set")
		j.Logger.Error(err, "no filename is set")

		return
	}

	dhcp.SetFilename(j.Logger, rep, filename, j.NextServer, isHTTPClient, httpPrefix)
}

// VLANID returns the VLAN ID for the job.
func (j *Job) VLANID() string {
	return j.hardware.GetVLANID(j.mac)
}
