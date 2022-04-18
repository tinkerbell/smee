package tinkerbell

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"

	"github.com/tinkerbell/boots/client"
)

func TestDiscoveryTinkerbell(t *testing.T) {
	for name, test := range tinkerbellTests {
		t.Run(name, func(t *testing.T) {
			d := DiscoveryTinkerbellV1{}

			if err := json.Unmarshal([]byte(test.json), &d); err != nil {
				t.Fatal(test.mode, err)
			}

			mac, err := net.ParseMAC(test.mac)
			if err != nil {
				t.Fatal("parse test mac", err)
			}

			// suggest we leave this here for the time being for ease of test why we're not seeing what we expect
			if d.ID != test.id {
				t.Fatalf("unexpected id, want: %s, got: %s", test.id, d.ID)
			}

			t.Logf("d.mac: %s", d.mac)
			t.Logf("dhcp %v", d.Network.InterfaceByMac(mac).DHCP)
			t.Log()
			if d.Network.InterfaceByMac(mac).DHCP.IP.Address.String() != test.ip.Address.String() {
				t.Fatalf("unexpectedclient.IP, want: %v, got: %v", test.ip, d.Network.InterfaceByMac(mac).DHCP.IP)
			}
			if d.Network.InterfaceByIp(test.ip.Address).DHCP.MAC.String() != mac.String() {
				t.Fatalf("unexpected mac, want: %s, got: %s", mac, d.Network.InterfaceByIp(test.ip.Address).DHCP.MAC)
			}
			if d.Network.InterfaceByMac(mac).DHCP.Hostname != test.hostname {
				t.Fatalf("unexpected hostname, want: %s, got: %s", test.hostname, d.Network.InterfaceByMac(mac).DHCP.Hostname)
			}
			if int(d.LeaseTime(mac).Seconds()) != test.leaseTime {
				t.Fatalf("unexpected lease time, want: %d, got: %d", test.leaseTime, d.LeaseTime(mac))
			}
			// note the difference between []string(nil) and []string{}; use cmp.Diff to check
			if !reflect.DeepEqual(d.Network.InterfaceByMac(mac).DHCP.NameServers, test.nameServers) {
				t.Fatalf("unexpected name servers, want: %v, got: %v", test.nameServers, d.Network.InterfaceByMac(mac).DHCP.NameServers)
			}
			if !reflect.DeepEqual(d.Network.InterfaceByMac(mac).DHCP.TimeServers, test.timeServers) {
				t.Fatalf("unexpected time servers, want: %v, got: %v", test.timeServers, d.Network.InterfaceByMac(mac).DHCP.TimeServers)
			}
			if d.Network.InterfaceByMac(mac).DHCP.IP.Gateway.String() != test.ip.Gateway.String() {
				t.Fatalf("unexpected gateway, want: %s, got: %v", test.ip.Gateway, d.Network.InterfaceByMac(mac).DHCP.IP.Gateway)
			}
			if d.Network.InterfaceByMac(mac).DHCP.Arch != test.arch {
				t.Fatalf("unexpected arch, want: %s, got: %s", test.arch, d.Network.InterfaceByMac(mac).DHCP.Arch)
			}
			if d.Network.InterfaceByMac(mac).DHCP.UEFI != test.uefi {
				t.Fatalf("unexpected uefi, want: %v, got: %v", test.uefi, d.Network.InterfaceByMac(mac).DHCP.UEFI)
			}

			t.Logf("netboot: %v", d.Network.InterfaceByMac(mac).Netboot)
			t.Logf("netboot allow_pxe: %v", d.Network.InterfaceByMac(mac).Netboot.AllowPXE)
			t.Logf("netboot allow_workflow: %v", d.Network.InterfaceByMac(mac).Netboot.AllowWorkflow)
			t.Logf("netbootclient.IPxe: %v", d.Network.InterfaceByMac(mac).Netboot.IPXE)
			t.Logf("netboot osie: %v", d.Network.InterfaceByMac(mac).Netboot.OSIE)
			t.Log()

			t.Logf("metadata: %v", d.Metadata)
			t.Logf("metadata state: %v", d.Metadata.State)
			t.Logf("metadata bonding_mode: %v", d.Metadata.BondingMode)
			t.Logf("metadata manufacturer: %v", d.Metadata.Manufacturer)
			if d.Instance() != nil {
				t.Logf("instance: %v", d.Instance())
				t.Logf("instance id: %s", d.Instance().ID)
				t.Logf("instance state: %s", d.Instance().State)

				if d.OperatingSystem().Distro != test.distro {
					t.Fatalf("unexpected os distro, want: %s, got: %s", test.distro, d.OperatingSystem().Distro)
				}

				d.OperatingSystem().Distro = "test" // test setting field inside operating system
				if d.OperatingSystem().Distro != "test" {
					t.Fatalf("could not set field inside operating system (distro), should be set to 'test', but got %s", d.OperatingSystem().Distro)
				}
			}
			t.Logf("metadata custom: %v", d.Metadata.Custom)
			t.Logf("metadata facility: %v", d.Metadata.Facility)

			t.Log()
			h := d.Hardware()
			for _, IP := range h.HardwareIPs() {
				t.Logf("hardwareclient.IP: %v", IP)
				t.Log()
			}

			d.SetMAC(mac)
			mode := d.Mode()
			if mode != test.mode {
				t.Fatalf("unexpected mode, want: %s, got: %s", test.mode, mode)
			}

			if mode == "" {
				return
			}

			conf := d.GetIP(mac)
			if conf.Address.String() != test.ip.Address.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.ip.Address, conf.Address)
			}
			if conf.Netmask.String() != test.ip.Netmask.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.ip.Netmask, conf.Netmask)
			}
			if conf.Gateway.String() != test.ip.Gateway.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.ip.Gateway, conf.Gateway)
			}
		})
	}
}

func TestOperatingSystemTinkerbell(t *testing.T) {
	for name, test := range tinkerbellTests {
		t.Run(name, func(t *testing.T) {
			d := &DiscoveryTinkerbellV1{}

			if err := json.Unmarshal([]byte(test.json), &d); err != nil {
				t.Fatal(test.mode, err)
			}

			if d.OperatingSystem().Distro != test.distro {
				t.Fatalf("unexpected instance operating system slug, want: %s, got: %s", test.distro, d.OperatingSystem().Distro)
			}
			d.OperatingSystem().Distro = "test" // test setting a field
			if d.OperatingSystem().Distro != "test" {
				t.Fatal("operating system slug should have been set to 'test'")
			}
		})
	}
}

var tinkerbellTests = map[string]struct {
	id            string
	mac           string
	ip            client.IP
	hostname      string
	leaseTime     int
	nameServers   []string
	timeServers   []string
	arch          string
	uefi          bool
	allowPXE      bool
	allowWorkflow bool
	ipxeURL       string
	ipxeContents  string
	mode          string
	osie          string
	distro        string
	json          string
}{
	"new_structure": {
		id:  "fde7c87c-d154-447e-9fce-7eb7bdec90c0",
		mac: "ec:0d:9a:c0:01:0c",
		ip: client.IP{
			Address: net.ParseIP("192.168.1.5"),
			Netmask: net.ParseIP("255.255.255.248"),
			Gateway: net.ParseIP("192.168.1.1"),
		},
		hostname:    "server001",
		leaseTime:   172801,
		nameServers: []string{},
		timeServers: []string{},
		arch:        "x86_64",
		uefi:        false,
		mode:        "hardware",
		json:        newJsonStruct,
	},
	"new_structure_defaults": {
		id:  "fde7c87c-d154-448e-9fce-7eb7bdec90c0",
		mac: "ec:0d:9a:c0:01:0d",
		ip: client.IP{
			Address: net.ParseIP("192.168.1.5"),
			Netmask: net.ParseIP("255.255.255.248"),
			Gateway: net.ParseIP("192.168.1.1"),
		},
		hostname:    "server001",
		leaseTime:   172800,
		nameServers: []string{"1.2.3.4"},
		timeServers: []string{},
		arch:        "x86_64",
		uefi:        false,
		mode:        "hardware",
		json:        newJsonStructUseDefaults,
	},
	"full structure tinkerbell": {
		id:  "0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94",
		mac: "00:00:00:00:00:00",
		ip: client.IP{
			Address: net.ParseIP("192.168.1.5"),
			Netmask: net.ParseIP("255.255.255.248"),
			Gateway: net.ParseIP("192.168.1.1"),
		},
		hostname:    "server001",
		leaseTime:   86400,
		nameServers: []string{},
		timeServers: []string{},
		arch:        "x86_64",
		uefi:        false,
		mode:        "hardware",
		distro:      "ubuntu",
		json:        fullStructTinkerbell,
	},
}

// use vim's (or equivalent) `!jq -S` on these strings
const (
	newJsonStructUseDefaults = `
{
  "id": "fde7c87c-d154-448e-9fce-7eb7bdec90c0",
  "network": {
    "interfaces": [
      {
        "dhcp": {
          "arch": "x86_64",
          "hostname": "server001",
          "ip": {
            "address": "192.168.1.5",
            "gateway": "192.168.1.1",
            "netmask": "255.255.255.248"
          },
          "mac": "ec:0d:9a:c0:01:0d",
          "name_servers": [
            "1.2.3.4"
          ],
          "time_servers": [],
          "uefi": false
        },
        "netboot": {
          "allow_pxe": true,
          "allow_workflow": true,
          "ipxe": {
            "contents": "#!ipxe",
            "url": "http://url/menu.ipxe"
          },
          "osie": {
            "base_url": "",
            "initrd": "",
            "kernel": "vmlinuz-x86_64"
          }
        }
      }
    ]
  }
}
`
	newJsonStruct = `
{
  "id": "fde7c87c-d154-447e-9fce-7eb7bdec90c0",
  "metadata": {
    "bonding_mode": 5,
    "custom": {
      "preinstalled_operating_system_version": {},
      "private_subnets": []
    },
    "facility": {
      "facility_code": "ewr1",
      "plan_slug": "c2.medium.x86",
      "plan_version_slug": ""
    },
    "instance": {},
    "manufacturer": {
      "id": "",
      "slug": ""
    },
    "state": ""
  },
  "network": {
    "interfaces": [
      {
        "dhcp": {
          "arch": "x86_64",
          "hostname": "server001",
          "ip": {
            "address": "192.168.1.5",
            "gateway": "192.168.1.1",
            "netmask": "255.255.255.248"
          },
          "lease_time": 172801,
          "mac": "ec:0d:9a:c0:01:0c",
          "name_servers": [],
          "time_servers": [],
          "uefi": false
        },
        "netboot": {
          "allow_pxe": true,
          "allow_workflow": true,
          "ipxe": {
            "contents": "#!ipxe",
            "url": "http://url/menu.ipxe"
          },
          "osie": {
            "base_url": "",
            "initrd": "",
            "kernel": "vmlinuz-x86_64"
          }
        }
      }
    ]
  }
}
 `
	fullStructTinkerbell = `
{
  "id": "0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94",
  "metadata": {
    "bonding_mode": 5,
    "custom": {
      "preinstalled_operating_system_version": {},
      "private_subnets": []
    },
    "facility": {
      "facility_code": "ewr1",
      "plan_slug": "c2.medium.x86",
      "plan_version_slug": ""
    },
    "instance": {
      "crypted_root_password": "redacted",
      "operating_system": {
        "distro": "ubuntu",
        "os_slug": "ubuntu_18_04",
        "version": "18.04"
      },
      "storage": {
        "disks": [
          {
            "device": "/dev/sda",
            "partitions": [
              {
                "label": "BIOS",
                "number": 1,
                "size": 4096
              },
              {
                "label": "SWAP",
                "number": 2,
                "size": 3993600
              },
              {
                "label": "ROOT",
                "number": 3,
                "size": 0
              }
            ],
            "wipe_table": true
          }
        ],
        "filesystems": [
          {
            "mount": {
              "create": {
                "options": [
                  "-L",
                  "ROOT"
                ]
              },
              "device": "/dev/sda3",
              "format": "ext4",
              "point": "/"
            }
          },
          {
            "mount": {
              "create": {
                "options": [
                  "-L",
                  "SWAP"
                ]
              },
              "device": "/dev/sda2",
              "format": "swap",
              "point": "none"
            }
          }
        ]
      }
    },
    "manufacturer": {
      "id": "",
      "slug": ""
    },
    "state": ""
  },
  "network": {
    "interfaces": [
      {
        "dhcp": {
          "arch": "x86_64",
          "hostname": "server001",
          "ip": {
            "address": "192.168.1.5",
            "gateway": "192.168.1.1",
            "netmask": "255.255.255.248"
          },
          "lease_time": 86400,
          "mac": "00:00:00:00:00:00",
          "name_servers": [],
          "time_servers": [],
          "uefi": false
        },
        "netboot": {
          "allow_pxe": true,
          "allow_workflow": true,
          "ipxe": {
            "contents": "#!ipxe",
            "url": "http://url/menu.ipxe"
          },
          "osie": {
            "base_url": "",
            "initrd": "",
            "kernel": "vmlinuz-x86_64"
          }
        }
      }
    ]
  }
}
`
)
