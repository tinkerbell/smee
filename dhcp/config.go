package dhcp

import (
	"net"
	"time"

	"github.com/go-logr/logr"
	dhcp4 "github.com/packethost/dhcp4-go"
	"github.com/pkg/errors"
)

type Config struct {
	addr net.IP
	opts dhcp4.OptionMap

	Log logr.Logger
}

func (c *Config) ApplyTo(rep *dhcp4.Packet) bool {
	if c.addr == nil {
		return false
	}

	rep.SetYIAddr(c.addr)
	for o, v := range c.opts {
		rep.SetOption(o, v)
	}

	return true
}

func (c *Config) Address() net.IP {
	return c.addr
}

func (c *Config) Netmask() net.IP {
	nm, ok := c.opts.GetIP(dhcp4.OptionSubnetMask)
	if !ok {
		return nil
	}

	return nm
}

func (c *Config) Gateway() net.IP {
	gw, ok := c.opts.GetIP(dhcp4.OptionRouter)
	if !ok {
		return nil
	}

	return gw
}

func (c *Config) Hostname() string {
	hn, ok := c.opts.GetString(dhcp4.OptionHostname)
	if !ok {
		return ""
	}

	return hn
}

func (c *Config) Setup(address, netmask, gateway net.IP) {
	if v4 := address.To4(); v4 != nil {
		c.addr = v4
		c.opts = make(dhcp4.OptionMap, 255)

		if netmask != nil {
			c.opts.SetIP(dhcp4.OptionSubnetMask, netmask)
		}

		if gateway != nil {
			c.opts.SetIP(dhcp4.OptionRouter, gateway)
		}
	} else {
		c.Log.Error(errors.New("address is not an IPv4 address"), "address is not an IPv4 address", "address", address)
		c.addr = nil
		c.opts = nil
	}
}

func (c *Config) SetLeaseTime(d time.Duration) {
	c.opts.SetDuration(dhcp4.OptionAddressTime, d)
}

func (c *Config) SetHostname(s string) {
	if s == "" {
		return
	}
	c.opts.SetString(dhcp4.OptionHostname, s)
}

func (c *Config) SetDHCPServer(ip net.IP) {
	v4 := ip.To4()
	if v4 == nil {
		c.Log.Error(errors.New("address is not an IPv4 address"), "address is not an IPv4 address", "address", ip)

		return
	}
	c.opts.SetOption(dhcp4.OptionDHCPServerID, []byte(v4))
}

func (c *Config) SetDNSServers(ips []net.IP) {
	if len(ips) == 0 {
		return
	}
	b := make([]byte, 0, 4*len(ips))
	for _, ip := range ips {
		v4 := ip.To4()
		if v4 == nil {
			c.Log.Info("skipping non IPv4 dns server address", "address", ip)

			continue
		}
		b = append(b, v4...)
	}
	if len(b) == 0 {
		c.Log.Error(errors.New("no IPv4 dns server address supplied"), "no IPv4 dns server address supplied")

		return
	}
	c.opts.SetOption(dhcp4.OptionDomainServer, b)
}

// SetOpt43SubOpt sets an option 43 sub-option. If option 43 is already set, the sub-option is appended.
func (c *Config) SetOpt43SubOpt(subOpt dhcp4.Option, s string) {
	if s == "" {
		return
	}
	n := make(dhcp4.OptionMap)
	// get current option 43
	cur, ok := c.opts.GetOption(dhcp4.OptionVendorSpecific)
	if ok {
		if err := n.Deserialize(cur, nil); err != nil {
			c.Log.Info("unable to deserialize option 43")

			return
		}
	}
	n.SetOption(subOpt, []byte(s))
	c.opts.SetOption(dhcp4.OptionVendorSpecific, n.Serialize())
}
