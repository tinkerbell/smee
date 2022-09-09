package cacher

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/tinkerbell/boots/client"
)

func TestDiscoveryCacher(t *testing.T) {
	for name, test := range cacherTests {
		t.Run(name, func(t *testing.T) {
			d := &DiscoveryCacher{}

			if err := json.Unmarshal([]byte(test.json), &d); err != nil {
				t.Fatal(test.mode, err)
			}

			mac, err := net.ParseMAC(test.mac)
			if err != nil {
				t.Fatal(test.mode, err)
			}

			// suggest we leave this here for the time being for ease of test why we're not seeing what we expect
			t.Logf("Discovery: %s", spew.Sdump(d))
			t.Logf("MacType: %s", d.MacType(mac.String()))
			t.Logf("MacIsType=data: %v", d.MacIsType(mac.String(), "data"))
			t.Logf("PrimaryDataMac: %s", d.PrimaryDataMAC().HardwareAddr().String())
			t.Logf("MAC: %v", d.MAC())
			t.Logf("mac: %s", mac.String())
			t.Logf("hardwareclient.IP: %v", d.hwIP())
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

			if d.PrimaryDataMAC().String() != test.primaryDataMac {
				t.Fatalf("unexpected address, want: %s, got: %s", test.primaryDataMac, d.PrimaryDataMAC().String())
			}
			if d.MAC().String() != test.mac {
				t.Fatalf("unexpected address, want: %s, got: %s", test.mac, d.MAC().String())
			}

			conf := d.GetIP(mac)
			if conf.Address.String() != test.conf.Address.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.conf.Address, conf.Address)
			}
			if conf.Netmask.String() != test.conf.Netmask.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.conf.Netmask, conf.Netmask)
			}
			if conf.Gateway.String() != test.conf.Gateway.String() {
				t.Fatalf("unexpected address, want: %s, got: %s", test.conf.Gateway, conf.Gateway)
			}

			osie := d.ServicesVersion.OSIE
			if osie != test.osie {
				t.Fatalf("unexpected osie version, want: %s, got: %s", test.osie, osie)
			}

			if d.GetTraceparent() != test.traceparent {
				t.Fatalf("unexpected traceparent, want: %s, got: %s", test.traceparent, d.GetTraceparent())
			}

			if d.Instance() != nil {
				if d.OperatingSystem().Distro != test.distro {
					t.Fatalf("unexpected distro, want: %s, got: %s", test.distro, d.OperatingSystem().Distro)
				}

				d.OperatingSystem().Distro = "test" // test setting field inside operating system
				if d.OperatingSystem().Distro != "test" {
					t.Fatalf("could not set field inside operating system (distro), should be set to 'test', but got %s", d.OperatingSystem().Distro)
				}
			}
		})
	}
}

func TestOperatingSystemCacher(t *testing.T) {
	for name, test := range cacherTests {
		t.Run(name, func(t *testing.T) {
			d := &DiscoveryCacher{}

			if err := json.Unmarshal([]byte(test.json), &d); err != nil {
				t.Fatal(test.mode, err)
			}

			if d.OperatingSystem().Distro != test.distro {
				t.Fatalf("unexpected instance operating system slug, want: %s, got: %s", test.distro, d.OperatingSystem().Distro)
			}
			d.OperatingSystem().Distro = "test" // test setting a field
			if d.OperatingSystem().Distro != "test" {
				t.Fatal("operating system distro should have been set to 'test'")
			}
		})
	}
}

var cacherTests = map[string]struct {
	mac            string
	primaryDataMac string
	mode           string
	conf           client.IP
	osie           string
	distro         string
	traceparent    string
	json           string
}{
	"unknown": {
		mac:            "84:b5:9c:cf:17:ff",
		primaryDataMac: "00:00:00:00:00:00",
		mode:           "",
		json:           discovered,
	},
	"discovered": {
		mac:            "84:b5:9c:cf:17:01",
		primaryDataMac: "00:00:00:00:00:00",
		mode:           "discovered",
		conf: client.IP{
			Address: net.ParseIP("10.250.142.74"),
			Gateway: net.ParseIP("10.250.142.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: discovered,
	},
	"management no instance": {
		mac:            "00:25:90:f6:2f:2d",
		primaryDataMac: "00:25:90:e7:6c:78",
		mode:           "management",
		conf: client.IP{
			Address: net.ParseIP("10.255.252.16"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: noInstance,
	},
	"no data macs": {
		mac:            "00:25:90:f6:2f:2d",
		primaryDataMac: "00:00:00:00:00:00",
		mode:           "management",
		conf: client.IP{
			Address: net.ParseIP("10.255.252.16"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: noMACData,
	},
	"management with instance": {
		mac:            "00:25:90:f6:28:5b",
		primaryDataMac: "00:25:90:e7:68:da",
		mode:           "management",
		conf: client.IP{
			Address: net.ParseIP("10.255.252.15"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		distro: "centos",
		json:   withInstance,
	},
	"instance": {
		mac:            "00:25:90:e7:68:da",
		primaryDataMac: "00:25:90:e7:68:da",
		mode:           "instance",
		conf: client.IP{
			Address: net.ParseIP("147.75.193.106"),
			Gateway: net.ParseIP("147.75.193.105"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		distro: "centos",
		json:   withInstance,
	},
	"preinstalling": {
		mac:            "00:25:99:e7:6c:78",
		primaryDataMac: "00:25:99:e7:6c:78",
		mode:           "hardware",
		conf: client.IP{
			Address: net.ParseIP("172.16.0.16"),
			Gateway: net.ParseIP("172.16.0.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: preinstalling,
	},
	"deprovisioning": {
		mac:            "fc:15:b4:97:04:e5",
		primaryDataMac: "fc:15:b4:97:04:e5",
		mode:           "instance",
		conf: client.IP{
			Address: net.ParseIP("172.16.0.14"),
			Gateway: net.ParseIP("172.16.0.13"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		distro: "centos",
		json:   deprovisioning,
	},
	"provisioning": {
		mac:            "fc:15:b4:97:04:f5",
		primaryDataMac: "fc:15:b4:97:04:f5",
		mode:           "instance",
		conf: client.IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		distro: "centos",
		json:   provisioning,
	},
	"provisioning with custom service": {
		mac:            "fc:15:b4:97:04:f5",
		primaryDataMac: "fc:15:b4:97:04:f5",
		mode:           "instance",
		osie:           "v19.01.01.00",
		conf: client.IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		distro: "centos",
		json:   provisioningWithService,
	},
	"full structure cacher": {
		mac:            "f4:79:2b:fa:4f:ae",
		primaryDataMac: "e4:45:19:c4:ba:50",
		mode:           "management",
		conf: client.IP{
			Address: net.ParseIP("10.255.3.13"),
			Gateway: net.ParseIP("10.255.3.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		distro:      "vmware",
		traceparent: "00-deadbeefcafedeadbeefcafedeadbeef-123456789abcdef0-01",
		json:        fullStructCacher,
	},
}

// use vim's (or equivalent) `!jq -S` on these strings.
const (
	discovered = `
{
  "id": "1a02e6c4-43e5-4be6-aa00-a8b42e4c770d",
  "management": {
    "address": "10.250.142.74",
    "gateway": "10.250.142.1",
    "netmask": "255.255.255.0"
  },
  "network_ports": [
    {
      "data": {
        "mac": "84:b5:9c:cf:17:01"
      },
      "name": "ipmi0",
      "type": "ipmi"
    }
  ]
}
`
	noMACData = `
{
  "arch": "x86_64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "d7e1feaf-d6d5-4d6c-8d16-5c6913be2dea",
  "instance": {},
  "ip_addresses": [
    {
      "address": "172.16.0.3",
      "address_family": 4,
      "enabled": true,
      "gateway": "172.16.0.2",
      "management": true,
      "netmask": "255.255.255.252",
      "public": false
    }
  ],
  "management": {
    "address": "10.255.252.16",
    "gateway": "10.255.252.1",
    "netmask": "255.255.255.0"
  },
  "manufacturer": {
    "id": "f7dbf901-d210-4594-ab82-f529a36bdd70",
    "slug": "supermicro"
  },
  "name": "sled5.mc1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "0b7fc8dc-33bf-4802-903f-55c4d076bfc7",
        "name": "ge-0/0/4",
        "type": "data"
      },
      "data": {
        "bond": "bond0"
      },
      "id": "179b020a-74ba-4969-a97a-f8e03b3877c8",
      "name": "eth0",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "216f9b60-6a99-460c-9e55-99becbd8776e",
        "name": "ge-1/0/4",
        "type": "data"
      },
      "data": {
        "bond": "bond0"
      },
      "id": "f2088273-a38f-4005-ba11-845e8e2aa342",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "c7389751-1699-4e17-ba1f-f1fb439aa666",
        "name": "Fa0/5",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "00:25:90:f6:2f:2d"
      },
      "id": "e6db1b93-f718-4bd1-9dea-05725a04a87a",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.small.x86",
  "state": "in_use",
  "type": "sled",
  "vlan_id": null
}
`
	noInstance = `
{
  "arch": "x86_64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "d7e1feaf-d6d5-4d6c-8d16-5c6913be2dea",
  "instance": {},
  "ip_addresses": [
    {
      "address": "172.16.0.3",
      "address_family": 4,
      "enabled": true,
      "gateway": "172.16.0.2",
      "management": true,
      "netmask": "255.255.255.252",
      "public": false
    }
  ],
  "management": {
    "address": "10.255.252.16",
    "gateway": "10.255.252.1",
    "netmask": "255.255.255.0"
  },
  "manufacturer": {
    "id": "f7dbf901-d210-4594-ab82-f529a36bdd70",
    "slug": "supermicro"
  },
  "name": "sled5.mc1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "0b7fc8dc-33bf-4802-903f-55c4d076bfc7",
        "name": "ge-0/0/4",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:90:e7:6c:78"
      },
      "id": "179b020a-74ba-4969-a97a-f8e03b3877c8",
      "name": "eth0",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "216f9b60-6a99-460c-9e55-99becbd8776e",
        "name": "ge-1/0/4",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:90:e7:6c:79"
      },
      "id": "f2088273-a38f-4005-ba11-845e8e2aa342",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "c7389751-1699-4e17-ba1f-f1fb439aa666",
        "name": "Fa0/5",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "00:25:90:f6:2f:2d"
      },
      "id": "e6db1b93-f718-4bd1-9dea-05725a04a87a",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.small.x86",
  "state": "in_use",
  "type": "sled",
  "vlan_id": null
}
`
	preinstalling = `
{
  "arch": "aarch64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "instance": {},
  "ip_addresses": [
    {
      "address": "172.16.0.16",
      "address_family": 4,
      "cidr": 30,
      "enabled": true,
      "gateway": "172.16.0.15",
      "management": true,
      "netmask": "255.255.255.252",
      "network": "172.16.0.14",
      "public": false,
      "type": "data"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "name": "sled3.arm1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "name": "xe-0/0/4:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:99:e7:6c:78"
      },
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "name": "eth0",
      "type": "data"
    },
    {
      "data": {
        "bond": null,
        "mac": "38:bc:01:c6:cc:de"
      },
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "name": "eth01",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "name": "xe-0/0/5:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:99:e7:6c:79"
      },
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "name": "Fa0/3",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "fc:15:b4:97:04:e7"
      },
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.large.arm",
  "preinstalled_operating_system_version": {
    "distro": "centos",
    "image_tag": null,
    "os_slug": "centos_7",
    "slug": "centos_7-t1.small.x86",
    "version": "7"
  },
  "state": "preinstalling",
  "type": "sled",
  "vlan_id": "122"
}
`
	withInstance = `
{
  "arch": "x86_64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "506ad180-8692-480d-b6c2-3ec7f8d719ac",
  "instance": {
    "allow_pxe": true,
    "always_pxe": false,
    "hostname": "test.smr.2",
    "id": "1d62730e-a7b6-4600-a424-17d26ccc1f59",
    "ip_addresses": [
      {
        "address": "147.75.193.106",
        "address_family": 4,
        "enabled": true,
        "gateway": "147.75.193.105",
        "management": true,
        "netmask": "255.255.255.252",
        "public": true
      }
    ],
    "ipxe_script_url": null,
    "operating_system_version": {
      "distro": "centos",
      "image_tag": null,
      "os_slug": "deprovision",
      "slug": "deprovision",
      "version": ""
    },
    "rescue": false,
    "ssh_keys": [],
    "state": "failed",
    "userdata": null
  },
  "ip_addresses": [
    {
      "address": "172.16.0.7",
      "address_family": 4,
      "enabled": true,
      "gateway": "172.16.0.6",
      "management": true,
      "netmask": "255.255.255.252",
      "public": false
    }
  ],
  "management": {
    "address": "10.255.252.15",
    "gateway": "10.255.252.1",
    "netmask": "255.255.255.0"
  },
  "manufacturer": {
    "id": "f7dbf901-d210-4594-ab82-f529a36bdd70",
    "slug": "supermicro"
  },
  "name": "sled4.mc1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "32480a81-d644-4226-a209-c600d9cc21d4",
        "name": "ge-0/0/3",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:90:e7:68:da"
      },
      "id": "3580ace2-1121-4ef9-8cd0-d471f8dc6fe5",
      "name": "eth0",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "5bd48979-f598-4e0a-969b-1e1bc8bd7284",
        "name": "ge-1/0/3",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "00:25:90:e7:68:db"
      },
      "id": "5415f58c-9f4a-4994-9f05-361fb646bce7",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "aeaec5b9-f820-4be4-abfb-3add725283a8",
        "name": "Fa0/4",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "00:25:90:f6:28:5b"
      },
      "id": "8c775f93-4ada-44ac-a9c7-6639bd3f3349",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.small.x86",
  "state": "in_use",
  "type": "sled",
  "vlan_id": null
}
`
	deprovisioning = `
{
  "arch": "aarch64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "instance": {
    "allow_pxe": true,
    "always_pxe": false,
    "hostname": "testing-layer-2",
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "ip_addresses": [],
    "ipxe_script_url": null,
    "operating_system_version": {
      "distro": "centos",
      "image_tag": null,
      "os_slug": "deprovision",
      "slug": "deprovision",
      "version": ""
    },
    "rescue": false,
    "ssh_keys": [],
    "state": "deprovisioning",
    "userdata": null
  },
  "ip_addresses": [
    {
      "address": "172.16.0.14",
      "address_family": 4,
      "cidr": 30,
      "enabled": true,
      "gateway": "172.16.0.13",
      "management": true,
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "public": false,
      "type": "data"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "name": "sled3.arm1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "name": "xe-0/0/4:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:e5"
      },
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "name": "eth0",
      "type": "data"
    },
    {
      "data": {
        "bond": null,
        "mac": "38:bc:01:c6:cc:de"
      },
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "name": "eth01",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "name": "xe-0/0/5:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:e6"
      },
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "name": "Fa0/3",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "fc:15:b4:97:04:e7"
      },
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.large.arm",
  "preinstalled_operating_system_version": {},
  "state": "deprovisioning",
  "type": "sled",
  "vlan_id": "122"
}
`
	provisioning = `
{
  "arch": "aarch64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "instance": {
    "allow_pxe": true,
    "always_pxe": false,
    "hostname": "testing-layer-2",
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "ip_addresses": [
      {
        "address": "147.75.14.16",
        "address_family": 4,
        "enabled": true,
        "gateway": "147.75.14.15",
        "management": true,
        "netmask": "255.255.255.252",
        "public": true
      }
    ],
    "ipxe_script_url": null,
    "operating_system_version": {
      "distro": "centos",
      "image_tag": null,
      "os_slug": "deprovision",
      "slug": "deprovision",
      "version": ""
    },
    "rescue": false,
    "ssh_keys": [],
    "state": "deprovisioning",
    "userdata": null
  },
  "ip_addresses": [
    {
      "address": "172.16.0.14",
      "address_family": 4,
      "cidr": 30,
      "enabled": true,
      "gateway": "172.16.0.13",
      "management": true,
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "public": false,
      "type": "data"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "name": "sled3.arm1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "name": "xe-0/0/4:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:f5"
      },
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "name": "eth0",
      "type": "data"
    },
    {
      "data": {
        "bond": null,
        "mac": "38:bc:01:c6:cc:de"
      },
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "name": "eth01",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "name": "xe-0/0/5:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:f6"
      },
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "name": "Fa0/3",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "fc:15:b4:97:04:e7"
      },
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.large.arm",
  "preinstalled_operating_system_version": {},
  "state": "deprovisioning",
  "type": "sled",
  "vlan_id": "122"
}
`
	provisioningWithService = `
{
  "arch": "aarch64",
  "bonding_mode": 4,
  "efi_boot": true,
  "facility_code": "lab1",
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "instance": {
    "allow_pxe": true,
    "always_pxe": false,
    "hostname": "testing-layer-2",
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "ip_addresses": [
      {
        "address": "147.75.14.16",
        "address_family": 4,
        "enabled": true,
        "gateway": "147.75.14.15",
        "management": true,
        "netmask": "255.255.255.252",
        "public": true
      }
    ],
    "ipxe_script_url": null,
    "operating_system_version": {
      "distro": "centos",
      "image_tag": null,
      "os_slug": "deprovision",
      "slug": "deprovision",
      "version": ""
    },
    "rescue": false,
    "ssh_keys": [],
    "state": "deprovisioning",
    "userdata": null
  },
  "ip_addresses": [
    {
      "address": "172.16.0.14",
      "address_family": 4,
      "cidr": 30,
      "enabled": true,
      "gateway": "172.16.0.13",
      "management": true,
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "public": false,
      "type": "data"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "name": "sled3.arm1.d11.lab1.packet.net",
  "network_ports": [
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "name": "xe-0/0/4:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:f5"
      },
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "name": "eth0",
      "type": "data"
    },
    {
      "data": {
        "bond": null,
        "mac": "38:bc:01:c6:cc:de"
      },
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "name": "eth01",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "name": "xe-0/0/5:2",
        "type": "data"
      },
      "data": {
        "bond": "bond0",
        "mac": "fc:15:b4:97:04:f6"
      },
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_port": {
        "data": {
          "bond": null,
          "mac": null
        },
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "name": "Fa0/3",
        "type": "data"
      },
      "data": {
        "bond": null,
        "mac": "fc:15:b4:97:04:e7"
      },
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "c1.large.arm",
  "preinstalled_operating_system_version": {},
  "services": {
    "osie": "v19.01.01.00"
  },
  "state": "deprovisioning",
  "type": "sled",
  "vlan_id": "122"
}
`
	fullStructCacher = `
{
  "allow_pxe": false,
  "arch": "x86_64",
  "bonding_mode": 4,
  "efi_boot": false,
  "facility_code": "dfw2",
  "id": "55639911-2278-498c-b364-8b2a62f5493c",
  "instance": {
    "allow_pxe": false,
    "always_pxe": false,
    "crypted_root_password": "r3d4c73d",
    "customdata": {},
    "hostname": "test",
    "id": "331d355a-1925-4ecf-ab6b-75990733c50c",
    "ip_addresses": [],
    "ipxe_script_url": null,
    "network_ready": false,
    "operating_system_version": {
      "distro": "vmware",
      "image_tag": "abc123",
      "os_slug": "vmware_esxi_7_0",
      "slug": "vmware_esxi_7_0",
      "version": "7.0"
    },
    "project": {
      "id": "b9638412-a71e-4390-8366-98bc6244725f",
      "name": "Test",
      "organization": {
        "id": "f27c556e-9476-4703-932f-a198b587c60d",
        "name": "Testing123"
      },
      "primary_owner": {
        "full_name": "Tester Jester",
        "id": "ca8c6279-1d01-4426-83de-cf7f0d62f0ed"
      }
    },
    "rescue": false,
    "ssh_keys": [],
    "state": "active",
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
              "size": "3993600"
            },
            {
              "label": "ROOT",
              "number": 3,
              "size": 0
            }
          ],
          "wipeTable": true
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
    },
    "tags": [],
    "userdata": ""
  },
  "ip_addresses": [
    {
      "address": "172.16.10.73",
      "address_family": 4,
      "cidr": 31,
      "enabled": true,
      "gateway": "172.16.10.72",
      "management": true,
      "netmask": "255.255.255.254",
      "network": "172.16.10.72",
      "port": "bond0",
      "public": false,
      "type": "data"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "manufacturer": {
    "id": "d2ea68db-a82a-4273-a590-594cf311ed52",
    "slug": "dell"
  },
  "name": "test.dfw2.packet.net",
  "network_ports": [
    {
      "connected_ports": [
        {
          "data": {
            "bond": null,
            "mac": null
          },
          "hostname": "test.test.dfw2.packet.net",
          "id": "dc76eb4f-6e21-4b6a-b622-08d1970b2224",
          "name": "xe-0/0/14:0",
          "type": "data"
        }
      ],
      "data": {
        "bond": "bond0",
        "mac": "e4:45:19:c4:ba:50"
      },
      "id": "c31fb859-1fe0-4a53-b42b-4cfb45fb7185",
      "name": "eth0",
      "type": "data"
    },
    {
      "connected_ports": [
        {
          "data": {
            "bond": null,
            "mac": null
          },
          "hostname": "test.test.dfw2.packet.net",
          "id": "a7e87953-0b44-4d4d-af63-2fac95083549",
          "name": "xe-0/0/15:0",
          "type": "data"
        }
      ],
      "data": {
        "bond": "bond1",
        "mac": "e4:45:19:c4:ba:51"
      },
      "id": "b676f474-5dc3-4337-8d57-b25b8f7febdf",
      "name": "eth1",
      "type": "data"
    },
    {
      "connected_ports": [
        {
          "data": {
            "bond": null,
            "mac": null
          },
          "hostname": "test.test.dfw2.packet.net",
          "id": "d10295d3-2bc6-4e76-afdc-e28383ba0766",
          "name": "xe-1/0/14:0",
          "type": "data"
        }
      ],
      "data": {
        "bond": "bond0",
        "mac": "e4:45:19:c4:ba:52"
      },
      "id": "36ba8732-05a4-46c6-857f-8e5765055394",
      "name": "eth2",
      "type": "data"
    },
    {
      "connected_ports": [
        {
          "data": {
            "bond": null,
            "mac": null
          },
          "hostname": "test.test.dfw2.packet.net",
          "id": "e8e9d7a2-bc7b-4a3d-9512-8f10b5a3632a",
          "name": "xe-1/0/15:0",
          "type": "data"
        }
      ],
      "data": {
        "bond": "bond1",
        "mac": "e4:45:19:c4:ba:53"
      },
      "id": "b36277ed-602d-4736-8fa3-fc1fa647e1d4",
      "name": "eth3",
      "type": "data"
    },
    {
      "connected_ports": [
        {
          "data": {
            "bond": null,
            "mac": null
          },
          "hostname": "rest.test.dfw2.packet.net",
          "id": "1fdcf403-59b3-40e5-a581-885d1349732e",
          "name": "ge-0/0/29",
          "type": "data"
        }
      ],
      "data": {
        "bond": null,
        "mac": "f4:79:2b:fa:4f:ae"
      },
      "id": "2f038e8a-9381-4dc0-8928-3a621a84fd09",
      "name": "ipmi0",
      "type": "ipmi"
    }
  ],
  "plan_slug": "n2.xlarge.x86",
  "plan_version_slug": "n2.xlarge.x86.01",
  "preinstalled_operating_system_version": {},
  "private_subnets": [
    "10.0.0.0/8"
  ],
  "services": {},
  "state": "in_use",
  "type": "server",
  "vlan_id": null,
  "traceparent": "00-deadbeefcafedeadbeefcafedeadbeef-123456789abcdef0-01"
}
`
)
