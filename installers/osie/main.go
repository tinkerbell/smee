package osie

import (
	"context"
	"strings"

	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/ipxe"
	"github.com/tinkerbell/boots/job"
	"go.opentelemetry.io/otel/trace"
)

type installer struct {
	osieURL string
	// defaultParams are passed to iPXE'd kernel always
	defaultParams string
	// workflowParams are passed to iPXE'd kernel when in tinkerbell or standalone mode and the hw indicates it can run workflows
	workflowParams      string
	osieFullURLOverride string
	extraIPXEVars       [][]string
}

// Installer instantiates a new osie installer.
func Installer(dataModelVersion, tinkGRPCAuth, extraKernelArgs, registry, registryUsername, registryPassword string, tinkTLS bool, osiePathOverride string, dynamicIPXEVars [][]string) job.BootScripter {
	defaultParams := []string{
		"ip=dhcp",
		"modules=loop,squashfs,sd-mod,usb-storage",
		"tinkerbell=${tinkerbell}",
		"syslog_host=${syslog_host}",
		"osie_vendors_url=" + conf.OsieVendorServicesURL,
	}

	if extraKernelArgs != "" {
		defaultParams = append(defaultParams, extraKernelArgs)
	}

	i := installer{
		osieURL:             conf.MirrorBaseURL + "/misc/osie",
		defaultParams:       strings.Join(defaultParams, " "),
		osieFullURLOverride: osiePathOverride,
		extraIPXEVars:       dynamicIPXEVars,
	}

	if dataModelVersion == "" {
		return i
	}

	workflowParams := []string{
		"grpc_authority=" + tinkGRPCAuth,
		"packet_base_url=" + conf.MirrorBaseURL + "/workflow",
	}
	if !tinkTLS {
		workflowParams = append(workflowParams, "tinkerbell_tls=false")
	}
	if registry != "" {
		workflowParams = append(workflowParams, "docker_registry="+registry)
	}
	if registryUsername != "" {
		workflowParams = append(workflowParams, "registry_username="+registryUsername)
	}
	if registryPassword != "" {
		workflowParams = append(workflowParams, "registry_password="+registryPassword)
	}
	i.workflowParams = strings.Join(workflowParams, " ")

	return i
}

func (i installer) BootScript(slug string) job.BootScript {
	switch slug {
	case "install", "rescue", "default":
		return i.install
	case "discover":
		return i.discover
	default:
		panic("unknown slug:" + slug)
	}
}

// install generates the ipxe boot script for booting into the osie installer.
func (i installer) install(ctx context.Context, j job.Job, s *ipxe.Script) {
	for _, kv := range i.extraIPXEVars {
		s.Set(kv[0], kv[1])
	}

	if j.Rescue() {
		i.rescue(ctx, j, s)

		return
	}

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

	i.setBootScript(ctx, "install", j, s)
}

// rescue generates the ipxe boot script for booting into osie in rescue mode.
func (i installer) rescue(ctx context.Context, j job.Job, s *ipxe.Script) {
	s.Set("action", "rescue")
	s.Set("state", j.HardwareState())
	i.setBootScript(ctx, "rescue", j, s)
}

func (i installer) discover(ctx context.Context, j job.Job, s *ipxe.Script) {
	for _, kv := range i.extraIPXEVars {
		s.Set(kv[0], kv[1])
	}

	s.Set("action", "discover")
	s.Set("state", j.HardwareState())

	i.setBootScript(ctx, "discover", j, s)
}

func (i installer) setBootScript(ctx context.Context, action string, j job.Job, s *ipxe.Script) {
	s.Set("arch", j.Arch())
	s.Set("bootdevmac", j.PrimaryNIC().String())
	s.Set("base-url", osieBaseURL(i.osieURL, i.osieFullURLOverride, j))
	s.Kernel("${base-url}/" + kernelPath(j))
	i.kernelParams(ctx, action, j.HardwareState(), j, s)
	s.Initrd("${base-url}/" + initrdPath(j))

	s.Boot()
}

func (i installer) kernelParams(ctx context.Context, action, _ string, j job.Job, s *ipxe.Script) {
	s.Args(i.defaultParams)

	// only add traceparent if tracing is enabled
	if sc := trace.SpanContextFromContext(ctx); sc.IsSampled() {
		// manually assemble a traceparent string because the "right" way is clunkier
		s.Args("traceparent=00-" + sc.TraceID().String() + "-" + sc.SpanID().String() + "-" + sc.TraceFlags().String())
	}

	if j.VLANID() != "" {
		s.Args("vlan_id=" + j.VLANID())
	}

	if j.CanWorkflow() {
		s.Args(i.workflowParams)
		s.Args("instance_id=" + j.InstanceID())
		s.Args("worker_id=" + j.HardwareID().String())
	} else {
		s.Args("packet_base_url=${base-url}")
	}

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
		// default serial console to ttyAMA0 for ARM.
		console = "ttyAMA0"
	} else {
		s.Args("console=tty0")
		// default serial console to ttyS1 for all other hardware.
		console = "ttyS1"
	}
	s.Args("console=" + console + ",115200")
}

func kernelPath(j job.Job) string {
	if path := j.KernelPath(); path != "" {
		return path
	}

	return "vmlinuz-${arch}"
}

func initrdPath(j job.Job) string {
	if path := j.InitrdPath(); path != "" {
		return path
	}

	return "initramfs-${arch}"
}

func isCustomOSIE(j job.Job) bool {
	return j.OSIEVersion() != ""
}

// osieBaseURL returns the value of Custom OSIE Service Version or just /current.
func osieBaseURL(osieURL string, osieFullURLOverride string, j job.Job) string {
	if osieFullURLOverride != "" {
		return osieFullURLOverride
	}
	if u := j.OSIEBaseURL(); u != "" {
		return u
	}
	if isCustomOSIE(j) {
		return osieURL + "/" + j.OSIEVersion()
	}

	return osieURL + "/current"
}
