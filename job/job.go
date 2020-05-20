package job

import (
	"fmt"
	"net"
	"time"

	"github.com/packethost/pkg/log"
	"github.com/pkg/errors"
	"github.com/tinkerbell/boots/conf"
	"github.com/tinkerbell/boots/dhcp"
	"github.com/tinkerbell/boots/packet"
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
	fmt.Println("ip in create from ip ", ip)
	mac := (*d).GetMac(ip)
	if mac.String() == packet.ZeroMAC.String() {
		joblog.With("ip", ip).Fatal(errors.New("somehow got a zero mac"))
	}
	j.mac = mac

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

func (j *Job) setup(dp *packet.Discovery) error {
	d := *dp
	dh := *d.Hardware()

	j.Logger = joblog.With("mac", j.mac, "hardware.id", dh.HardwareID())

	// mac is needed to find the hostname for DiscoveryCacher
	d.SetMac(j.mac)

	// dh.ID()
	// is this necessary?
	j.hardware = d.Hardware()

	//how can we remove this?
	j.instance = d.Instance()
	if j.instance == nil {
		j.instance = &packet.Instance{}
	} else {
		j.Logger = j.Logger.With("instance.id", j.InstanceID())
	}

	ip := d.GetIp(j.mac)
	if ip.Address == nil {
		return errors.New("could not find IP address")
	}
	j.dhcp.Setup(ip.Address, ip.Netmask, ip.Gateway)
	j.dhcp.SetLeaseTime(d.LeaseTime(j.mac))    // cacher=env.DHCPLeaseTime
	j.dhcp.SetDHCPServer(conf.PublicIPv4) // used for the unicast DHCPREQUEST
	j.dhcp.SetDNSServers(d.DnsServers())  // cacher=env.DNSServers

	hostname, err := d.Hostname()
	if err != nil {
		return err
	}
	if hostname != "" {
		j.dhcp.SetHostname(hostname)
	}

	return nil
}
