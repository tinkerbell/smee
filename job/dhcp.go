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

		isOuriPXE := ipxe.IsOuriPXE(req)
		if isOuriPXE {
			ipxe.Setup(rep)
		}

		filename := j.getPXEFilename(isOuriPXE, isARM, isUEFI)
		if filename == "" {
			err := errors.New("no filename is set")
			j.Error(err)

			return false
		}
		dhcp.SetFilename(rep, filename, conf.PublicIPv4)

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

func (j Job) getPXEFilename(isOuriPXE, isARM, isUEFI bool) string {
	if !j.isPXEAllowed() {
		if j.instance != nil && j.instance.State == "active" {
			// We set a filename because if a machine is actually trying to PXE and nothing is sent it may hang for
			// a while waiting for any possible ProxyDHCP packets and it would delay booting from disks.
			// This short cuts all that when we know we want to be booting from disk.
			return "/pxe-is-not-allowed"
		}

		return ""
	}

	var filename string
	if !isOuriPXE {
		if j.PArch() == "hua" || j.PArch() == "2a2" {
			filename = "snp-hua.efi"
		} else if isARM {
			filename = "snp-nolacp.efi"
		} else if isUEFI {
			filename = "ipxe.efi"
		} else {
			filename = "undionly.kpxe"
		}
	} else {
		filename = "http://" + conf.PublicFQDN + "/auto.ipxe"
	}

	return filename
}
