package osie

import (
	"context"
	"strings"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
	"go.opentelemetry.io/otel/trace"
)

type Installer struct{}

func (i Installer) Rescue() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.Set("action", "rescue")
		s.Set("state", j.HardwareState())

		return bootScript(ctx, "rescue", j, s)
	}
}

func (i Installer) Install() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		typ := "provisioning.104.01"
		if j.HardwareState() == "deprovisioning" {
			typ = "deprovisioning.304.1"
		}
		s.PhoneHome(typ)
		if j.CanWorkflow() {
			s.Set("action", "workflow")
		} else {
			s.Set("action", "install")
		}
		s.Set("state", j.HardwareState())

		return bootScript(ctx, "install", j, s)
	}
}

func (i Installer) Discover() job.BootScript {
	return func(ctx context.Context, j job.Job, s ipxe.Script) ipxe.Script {
		s.Set("action", "discover")
		s.Set("state", j.HardwareState())

		return bootScript(ctx, "discover", j, s)
	}
}

func bootScript(ctx context.Context, action string, j job.Job, s ipxe.Script) ipxe.Script {
	s.Set("arch", j.Arch())
	s.Set("parch", j.PArch())
	s.Set("bootdevmac", j.PrimaryNIC().String())
	s.Set("base-url", osieBaseURL(j))
	s.Kernel("${base-url}/" + kernelPath(j))

	ks := kernelParams(ctx, action, j.HardwareState(), j, s)

	ks.Initrd("${base-url}/" + initrdPath(j))

	if j.PArch() == "hua" || j.PArch() == "2a2" {
		// Workaround for Huawei firmware crash
		ks.Sleep(15)
	}

	ks.Boot()

	return ks
}

func kernelParams(ctx context.Context, action, state string, j job.Job, s ipxe.Script) ipxe.Script {
	s.Args("ip=dhcp") // Dracut?
	s.Args("modules=loop,squashfs,sd-mod,usb-storage")
	s.Args("alpine_repo=" + alpineMirror(j))
	s.Args("modloop=${base-url}/" + modloopPath(j))
	s.Args("tinkerbell=${tinkerbell}")
	s.Args("syslog_host=${syslog_host}")
	s.Args("parch=${parch}")
	s.Args("packet_action=${action}")
	s.Args("packet_state=${state}")

	// only add traceparent if tracing is enabled
	if sc := trace.SpanContextFromContext(ctx); sc.IsSampled() {
		// manually assemble a traceparent string because the "right" way is clunkier
		s.Args("traceparent=00-" + sc.TraceID().String() + "-" + sc.SpanID().String() + "-" + sc.TraceFlags().String())
	}

	// Only provide the Hollow secrets for deprovisions
	if j.HardwareState() == "deprovisioning" && conf.HollowClientId != "" && conf.HollowClientRequestSecret != "" {
		s.Args("hollow_client_id=" + conf.HollowClientId)
		s.Args("hollow_client_request_secret=" + conf.HollowClientRequestSecret)
	}

	if isCustomOSIE(j) {
		s.Args("packet_base_url=" + osieBaseURL(j))
	}

	if j.CanWorkflow() {
		buildWorkerParams()
		s.Args("docker_registry=" + dockerRegistry)
		s.Args("grpc_authority=" + grpcAuthority)
		s.Args("grpc_cert_url=" + grpcCertURL)
		s.Args("instance_id=" + j.InstanceID())
		s.Args("registry_username=" + registryUsername)
		s.Args("registry_password=" + registryPassword)
		s.Args("packet_base_url=" + workflowBaseURL())
		s.Args("worker_id=" + j.HardwareID().String())
	}

	s.Args("packet_bootdev_mac=${bootdevmac}")
	s.Args("facility=" + j.FacilityCode())

	switch j.PlanSlug() {
	case "c2.large.arm", "c2.large.anbox", "c3.large.arm":
		s.Args("iommu.passthrough=1")
	default:
		s.Args("intel_iommu=on iommu=pt")
	}

	if action == "install" {
		s.Args("plan=" + j.PlanSlug())
		s.Args("manufacturer=" + j.Manufacturer())

		slug := strings.TrimSuffix(j.OperatingSystem().OsSlug, "_image")
		tag := j.OperatingSystem().ImageTag

		if len(tag) > 0 {
			s.Args("slug=" + slug + ":" + tag)
		} else {
			s.Args("slug=" + slug)
		}

		if j.PasswordHash() != "" {
			s.Args("pwhash=" + j.PasswordHash())
		}
	}

	s.Args("initrd=" + initrdPath(j))

	var console string
	if j.IsARM() {
		console = "ttyAMA0"
		if j.PlanSlug() == "baremetal_hua" {
			console = "ttyS0"
		}
	} else {
		s.Args("console=tty0")
		if j.PlanSlug() == "d1p.optane.x86" || j.PlanSlug() == "d1f.optane.x86" || j.PlanSlug() == "w3amd.75xx24c.256.4320" {
			console = "ttyS0"
		} else {
			console = "ttyS1"
		}
	}
	s.Args("console=" + console + ",115200")

	return s
}

func alpineMirror(j job.Job) string {
	return "${base-url}/repo-${arch}/main"
}

func modloopPath(j job.Job) string {
	return "modloop-${parch}"
}

func kernelPath(j job.Job) string {
	if j.KernelPath() != "" {
		return j.KernelPath()
	}

	return "vmlinuz-${parch}"
}

func initrdPath(j job.Job) string {
	if j.InitrdPath() != "" {
		return j.InitrdPath()
	}

	return "initramfs-${parch}"
}

func isCustomOSIE(j job.Job) bool {
	return j.OSIEVersion() != ""
}

// osieBaseURL returns the value of Custom OSIE Service Version or just /current
func osieBaseURL(j job.Job) string {
	if u := j.OSIEBaseURL(); u != "" {
		return u
	}
	if isCustomOSIE(j) {
		return osieURL + "/" + j.OSIEVersion()
	}

	return osieURL + "/current"
}

func workflowBaseURL() string {
	return mirrorBaseURL + "/workflow"
}
