package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tinkerbell/boots/packet"
)

func main() {
	var m map[string]interface{}
	err := json.NewDecoder(os.Stdin).Decode(&m)
	if err != nil {
		panic(err)
	}
	instance := m["instance"].(map[string]interface{})

	buf := bytes.NewBuffer(nil)
	err = json.NewEncoder(buf).Encode(m)
	if err != nil {
		panic(err)
	}

	c := packet.DiscoveryCacher{}
	err = json.NewDecoder(buf).Decode(&c)
	if err != nil {
		panic(err)
	}

	d := packet.HardwareTinkerbellV1{
		ID: c.ID,
		Network: packet.Network{
			Interfaces: func() []packet.NetworkInterface {
				ifaces := make([]packet.NetworkInterface, 0, len(c.NetworkPorts))
				pmac := c.PrimaryDataMAC()
				var pip packet.IP
				for _, ip := range c.IPs {
					if ip.Family == 4 && ip.Management == true {
						pip = ip
						break
					}
				}
				for _, p := range c.NetworkPorts {
					ni := packet.NetworkInterface{
						DHCP: packet.DHCP{
							Arch:      c.Arch,
							IfaceName: p.Name,
							MAC:       p.Data.MAC,
						},
					}
					if *ni.DHCP.MAC == pmac {
						ni.DHCP.IP = pip
						ni.Netboot.AllowPXE = c.AllowPXE
						ni.Netboot.AllowWorkflow = true
					}
					ifaces = append(ifaces, ni)
				}
				return ifaces
			}(),
		},
		Metadata: packet.Metadata{
			State:        c.State,
			BondingMode:  c.BondingMode,
			Manufacturer: c.Manufacturer,
			Facility: packet.Facility{
				PlanSlug:        c.PlanSlug,
				PlanVersionSlug: c.PlanVersionSlug,
				FacilityCode:    c.FacilityCode,
			},
		},
	}
	d.Metadata.Custom.PreinstalledOS = c.PreinstallOS
	d.Metadata.Custom.PrivateSubnets = c.PrivateSubnets

	// populate metadata.instance straight from cacher version as boot's Instance struct doesn't have all the attributes we care about
	b, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}
	metadata := m["metadata"].(map[string]interface{})
	metadata["instance"] = instance
	m["metadata"] = metadata

	err = json.NewEncoder(os.Stdout).Encode(m)
	if err != nil {
		panic(err)
	}

	fmt.Println()
}
