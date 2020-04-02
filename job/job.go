package job

import (
	"net"
	"time"

	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/env"
	"github.com/tinkerbell/boots/packet"
	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
)

var client *packet.Client

// SetClient sets the client used to interact with the api.
func SetClient(c *packet.Client) {
	client = c
}

// Job this comment is useless
type Job struct {
	log.Logger

	mac net.HardwareAddr
	ip  net.IP

	start time.Time

	mode Mode
	dhcp dhcp.Config

	hardware *packet.Hardware
	instance *packet.Instance
}

// CreateFromDHCP looks up hardware using the MAC from cacher to create a job
func CreateFromDHCP(mac net.HardwareAddr, giaddr net.IP, circuitID string) (Job, error) {
	j := Job{
		mac:   mac,
		start: time.Now(),
	}

	d, err := discoverHardwareFromDHCP(mac, giaddr, circuitID)
	if err != nil {
		return Job{}, errors.WithMessage(err, "discover from dhcp message")
	}

	err = j.setup(d)
	if err != nil {
		return Job{}, err
	}
	return j, nil
}

// CreateFromRemoteAddr looks up hardware using the IP from cacher to create a job
func CreateFromRemoteAddr(ip string) (Job, error) {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		return Job{}, errors.Wrap(err, "splitting host:ip")
	}
	return CreateFromIP(net.ParseIP(host))
}

// CreateFromIP looksup hardware using the IP from cacher to create a job
func CreateFromIP(ip net.IP) (Job, error) {
	j := Job{
		ip:    ip,
		start: time.Now(),
	}

	joblog.With("ip", ip).Info("discovering from ip")
	d, err := discoverHardwareFromIP(ip)
	if err != nil {
		return Job{}, errors.WithMessage(err, "discovering from ip address")
	}
	mac := d.PrimaryDataMAC()
	if mac.IsZero() {
		joblog.With("ip", ip).Fatal(errors.New("somehow got a zero mac"))
	}
	j.mac = mac.HardwareAddr()

	err = j.setup(d)
	if err != nil {
		return Job{}, err
	}
	return j, nil
}

func (j Job) MarkDeviceActive() {
	if id := j.InstanceID(); id != "" {
		if err := client.PostInstancePhoneHome(id); err != nil {
			j.Error(err)
		}
	}
}

// golangci-lint: unused
//func (j Job) markFailed(reason string) {
//	j.postFailure(reason)
//}

func (j *Job) setup(d *packet.Discovery) error {
	mode, netConfig := d.Mode(j.mac), d.NetConfig(j.mac)
	j.Logger = joblog.With("mac", j.mac, "hardware.id", d.Hardware.ID)

	if netConfig.Address == nil {
		return errors.New("did not find a usable ip address in cacher data")
	}

	j.Logger = j.Logger.With("ip", netConfig.Address)

	j.hardware = d.Hardware
	j.instance = d.Instance
	if j.instance == nil {
		j.instance = &packet.Instance{}
	} else {
		j.Logger = j.Logger.With("instance.id", j.InstanceID())
	}

	j.Logger.With("mode", mode).Info("job setup configured")

	j.dhcp.Setup(netConfig.Address, netConfig.Netmask, netConfig.Gateway)
	j.dhcp.SetLeaseTime(env.DHCPLeaseTime)
	j.dhcp.SetDHCPServer(env.PublicIPv4) // used for the unicast DHCPREQUEST
	j.dhcp.SetDNSServers(env.DNSServers)

	var hostname string
	switch mode {
	case "discovered", "management":
		j.mode = modeManagement
		hostname = d.Hardware.Name
	case "instance":
		i := j.instance

		hostname = i.Hostname
		j.mode = modeInstance
		switch j.HardwareState() {
		case "deprovisioning":
			j.mode = modeDeprov
			hostname = d.Hardware.Name
		case "provisioning":
			j.mode = modeProv
		}
	case "hardware":
		j.mode = modeHardware
		hostname = d.Hardware.Name
	default:
		return errors.Errorf("unknown mode: %s", mode)
	}

	if hostname != "" {
		j.dhcp.SetHostname(hostname)
	}

	return nil
}
