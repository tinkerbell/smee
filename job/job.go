package job

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/packet"
	tw "github.com/tinkerbell/tink/protos/workflow"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var client packet.Client
var provisionerEngineName string

// SetClient sets the client used to interact with the api.
func SetClient(c packet.Client) {
	client = c
}

// SetProvisionerEngineName sets the provisioning engine name used
// for this instance of boots
func SetProvisionerEngineName(engineName string) {
	provisionerEngineName = engineName
}

// Job holds per request data
type Job struct {
	log.Logger
	mac      net.HardwareAddr
	ip       net.IP
	start    time.Time
	mode     Mode
	dhcp     dhcp.Config
	hardware packet.Hardware
	instance *packet.Instance
}

type Installers struct {
	Default     BootScript
	ByInstaller map[string]BootScript
	ByDistro    map[string]BootScript
	BySlug      map[string]BootScript
}

func NewInstallers() Installers {
	return Installers{
		Default:     nil,
		ByInstaller: make(map[string]BootScript),
		ByDistro:    make(map[string]BootScript),
		BySlug:      make(map[string]BootScript),
	}
}

// AllowPxe returns the value from the hardware data
// in tink server defined at network.interfaces[].netboot.allow_pxe
func (j Job) AllowPxe() bool {
	return j.hardware.HardwareAllowPXE(j.mac)
}

// ProvisionerEngineName returns the current provisioning engine name
// as defined by the env var PROVISIONER_ENGINE_NAME supplied at runtime
func (j Job) ProvisionerEngineName() string {
	return provisionerEngineName
}

// HasActiveWorkflow fetches workflows for the given hardware and returns
// the status true if there is a pending (active) workflow
func HasActiveWorkflow(ctx context.Context, hwID packet.HardwareID) (bool, error) {
	wcl, err := client.GetWorkflowsFromTink(ctx, hwID)
	if err != nil {
		return false, err
	}
	for _, wf := range (*wcl).WorkflowContexts {
		if wf.CurrentActionState == tw.State_STATE_PENDING || wf.CurrentActionState == tw.State_STATE_RUNNING {
			joblog.With("workflowID", wf.WorkflowId).Info("found active workflow for hardware")

			return true, nil
		}
	}

	return false, nil
}

// CreateFromDHCP looks up hardware using the MAC from cacher to create a job
func CreateFromDHCP(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (Job, error) {
	j := Job{
		mac:   mac,
		start: time.Now(),
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent("discoverHardwareFromDHCP",
		trace.WithAttributes(attribute.String("MAC", mac.String())),
		trace.WithAttributes(attribute.String("IP", giaddr.String())),
		trace.WithAttributes(attribute.String("CircuitID", circuitID)),
	)

	d, err := discoverHardwareFromDHCP(ctx, mac, giaddr, circuitID)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return Job{}, errors.WithMessage(err, "discover from dhcp message")
	}

	err = j.setup(d)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		j = Job{} // return an empty job on error
	}

	return j, err
}

// CreateFromRemoteAddr looks up hardware using the IP from cacher to create a job
func CreateFromRemoteAddr(ctx context.Context, ip string) (Job, error) {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		return Job{}, errors.Wrap(err, "splitting host:ip")
	}

	return CreateFromIP(ctx, net.ParseIP(host))
}

// CreateFromIP looksup hardware using the IP from cacher to create a job
func CreateFromIP(ctx context.Context, ip net.IP) (Job, error) {
	j := Job{
		ip:    ip,
		start: time.Now(),
	}

	joblog.With("ip", ip).Info("discovering from ip")
	d, err := discoverHardwareFromIP(ctx, ip)
	if err != nil {
		return Job{}, errors.WithMessage(err, "discovering from ip address")
	}
	mac := d.GetMAC(ip)
	if mac.String() == packet.ZeroMAC.String() {
		joblog.With("ip", ip).Fatal(errors.New("somehow got a zero mac"))
	}
	j.mac = mac

	err = j.setup(d)
	if err != nil {
		return Job{}, err
	}

	if os.Getenv("DATA_MODEL_VERSION") != "1" {
		return j, nil
	}

	hd := d.Hardware()
	hwID := hd.HardwareID()

	joblog.With("hardwareID", hwID).Info("fetching workflows for hardware")
	activeWorkflows, err := HasActiveWorkflow(ctx, hwID)
	if err != nil {
		return Job{}, err
	}

	if !activeWorkflows {
		return Job{}, errors.Errorf("no active workflow found for hardware %s", hwID)
	}

	return j, nil
}

// MarkDeviceActive marks the device active
func (j Job) MarkDeviceActive(ctx context.Context) {
	if id := j.InstanceID(); id != "" {
		if err := client.PostInstancePhoneHome(ctx, id); err != nil {
			j.Error(err)
		}
	}
}

func (j *Job) setup(d packet.Discovery) error {
	dh := d.Hardware()

	j.Logger = joblog.With("mac", j.mac, "hardware.id", dh.HardwareID())

	// mac is needed to find the hostname for DiscoveryCacher
	d.SetMAC(j.mac)

	// (kdeng3849) is this necessary?
	j.hardware = d.Hardware()

	// (kdeng3849) how can we remove this?
	j.instance = d.Instance()
	if j.instance == nil {
		j.instance = &packet.Instance{}
	} else {
		j.Logger = j.Logger.With("instance.id", j.InstanceID())
	}

	ip := d.GetIP(j.mac)
	if ip.Address == nil {
		return errors.New("could not find IP address")
	}
	j.dhcp.Setup(ip.Address, ip.Netmask, ip.Gateway)
	j.dhcp.SetLeaseTime(d.LeaseTime(j.mac))
	j.dhcp.SetDHCPServer(conf.PublicIPv4) // used for the unicast DHCPREQUEST
	j.dhcp.SetDNSServers(d.DnsServers(j.mac))

	hostname, err := d.Hostname()
	if err != nil {
		return err
	}
	if hostname != "" {
		j.dhcp.SetHostname(hostname)
	}

	return nil
}
