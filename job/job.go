package job

import (
	"context"
	"net"
	"time"

	"github.com/equinix-labs/otel-init-go/otelhelpers"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"go.opentelemetry.io/otel/trace"
)

// JobManager creates jobs.
type Manager interface {
	CreateFromRemoteAddr(ctx context.Context, ip string) (context.Context, *Job, error)
	CreateFromDHCP(context.Context, net.HardwareAddr, net.IP, string) (context.Context, *Job, error)
}

// Creator is a type that can create jobs.
type Creator struct {
	finder                client.HardwareFinder
	provisionerEngineName string
	logger                log.Logger
}

// NewCreator returns a manager that can create jobs.
func NewCreator(logger log.Logger, provisionerEngineName string, finder client.HardwareFinder) *Creator {
	return &Creator{
		finder:                finder,
		provisionerEngineName: provisionerEngineName,
		logger:                logger,
	}
}

var joblog log.Logger

func Init(l log.Logger) {
	joblog = l.Package("job")
	initRSA()
}

// Job holds per request data.
type Job struct {
	log.Logger
	provisionerEngineName string
	mac                   net.HardwareAddr
	ip                    net.IP
	start                 time.Time
	dhcp                  dhcp.Config
	hardware              client.Hardware
	instance              *client.Instance
	NextServer            net.IP
	IpxeBaseURL           string
	BootsBaseURL          string
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
// in tink server defined at network.interfaces[].netboot.allow_pxe.
func (j Job) AllowPXE() bool {
	if j.hardware.HardwareAllowPXE(j.mac) {
		return true
	}
	if j.InstanceID() == "" {
		return false
	}

	return j.instance.AllowPXE
}

// ProvisionerEngineName returns the current provisioning engine name
// as defined by the env var PROVISIONER_ENGINE_NAME supplied at runtime.
func (j Job) ProvisionerEngineName() string {
	return j.provisionerEngineName
}

// CreateFromDHCP looks up hardware using the MAC from cacher to create a job.
// OpenTelemetry: If a hardware record is available and has an in-band traceparent
// specified, the returned context will have that trace set as its parent and the
// spans will be linked.
func (c *Creator) CreateFromDHCP(ctx context.Context, mac net.HardwareAddr, giaddr net.IP, circuitID string) (context.Context, *Job, error) {
	j := &Job{
		mac:                   mac,
		start:                 time.Now(),
		provisionerEngineName: c.provisionerEngineName,
		Logger:                c.logger,
	}
	d, err := c.finder.ByMAC(ctx, mac, giaddr, circuitID)
	if err != nil {
		return ctx, nil, errors.WithMessage(err, "discover from dhcp message")
	}

	newCtx, err := j.setup(ctx, d)
	if err != nil {
		return ctx, nil, err
	}

	return newCtx, j, nil
}

// CreateFromRemoteAddr looks up hardware using the IP from cacher to create a job.
// OpenTelemetry: If a hardware record is available and has an in-band traceparent
// specified, the returned context will have that trace set as its parent and the
// spans will be linked.
func (c *Creator) CreateFromRemoteAddr(ctx context.Context, ip string) (context.Context, *Job, error) {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "splitting host:ip")
	}

	return c.createFromIP(ctx, net.ParseIP(host))
}

// createFromIP looks up hardware using the IP from cacher to create a job.
// OpenTelemetry: If a hardware record is available and has an in-band traceparent
// specified, the returned context will have that trace set as its parent and the
// spans will be linked.
func (c *Creator) createFromIP(ctx context.Context, ip net.IP) (context.Context, *Job, error) {
	j := &Job{
		ip:                    ip,
		start:                 time.Now(),
		provisionerEngineName: c.provisionerEngineName,
		Logger:                c.logger,
	}

	c.logger.With("ip", ip).Info("discovering from ip")
	d, err := c.finder.ByIP(ctx, ip)
	if err != nil {
		return ctx, nil, errors.WithMessage(err, "discovering from ip address")
	}
	mac := d.GetMAC(ip)
	if mac.String() == client.MinMAC.String() {
		c.logger.With("ip", ip).Fatal(errors.New("somehow got a zero mac"))
	}
	j.mac = mac

	ctx, err = j.setup(ctx, d)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, j, nil
}

// setup initializes the job from the discovered hardware record with the DHCP
// settings filled in from that record. If the inbound discovered hardware
// has an in-band traceparent populated, the context has its trace modified
// so that it points at the incoming traceparent from the hardware. A span
// link is applied in the process. The returned context's parent trace will
// be set to the traceparent value.
func (j *Job) setup(ctx context.Context, d client.Discoverer) (context.Context, error) {
	dh := d.Hardware()

	j.Logger = j.Logger.With("mac", j.mac, "hardware.id", dh.HardwareID())

	// When there is a traceparent in the hw record, create a link on the current
	// trace and replace ctx with one that is parented to the traceparent.
	if dh.GetTraceparent() != "" {
		fromLink := trace.LinkFromContext(ctx)
		ctx = otelhelpers.ContextWithTraceparentString(ctx, dh.GetTraceparent())
		trace.WithLinks(fromLink, trace.LinkFromContext(ctx))
	}

	// mac is needed to find the hostname for DiscoveryCacher
	d.SetMAC(j.mac)

	// (kdeng3849) is this necessary?
	j.hardware = d.Hardware()

	// (kdeng3849) how can we remove this?
	j.instance = d.Instance()
	if j.instance == nil {
		j.instance = &client.Instance{}
	} else {
		j.Logger = j.Logger.With("instance.id", j.InstanceID())
	}

	ip := d.GetIP(j.mac)
	if ip.Address == nil {
		return ctx, errors.New("could not find IP address")
	}
	j.dhcp.Setup(ip.Address, ip.Netmask, ip.Gateway)
	j.dhcp.SetLeaseTime(d.LeaseTime(j.mac))
	j.dhcp.SetDHCPServer(conf.PublicIPv4) // used for the unicast DHCPREQUEST
	j.dhcp.SetDNSServers(d.DNSServers(j.mac))

	hostname, err := d.Hostname()
	if err != nil {
		return ctx, err
	}
	if hostname != "" {
		j.dhcp.SetHostname(hostname)
	}

	// set option 43.116 to vlan id. If dh.GetVLANID is "", then j.dhcp.SetOpt43SubOpt is a no-op.
	j.dhcp.SetOpt43SubOpt(116, dh.GetVLANID(j.mac))

	return ctx, nil
}
