package packet

import (
	"encoding/json"
	"net"
	"testing"
)

func TestDiscovery(t *testing.T) {
	for name, test := range tests {
		t.Log(name)
		d := Discovery{}
		if err := json.Unmarshal([]byte(test.json), &d); err != nil {
			t.Fatal(test.mode, err)
		}

		mac, err := net.ParseMAC(test.mac)
		if err != nil {
			t.Fatal(test.mode, err)
		}

		// suggest we leave this here for the time being for ease of test why we're not seeing what we expect
		t.Logf("macType: %s\n", d.macType(mac.String()))
		t.Logf("macIsType=data: %v\n", d.macIsType(mac.String(), "data"))
		t.Logf("primaryDataMac: %s\n", d.PrimaryDataMAC().HardwareAddr().String())
		t.Logf("mac: %s\n", mac.String())
		if d.Instance != nil {
			t.Logf("instance: %v\n", d.Instance)
			t.Logf("instanceId: %s\n", d.Instance.ID)
			t.Logf("instance State: %s\n", d.Instance.State)
		}
		t.Logf("hardware IP: %v\n", d.hardwareIP())
		t.Log("\n")
		for _, ip := range d.Hardware.IPs {
			t.Logf("hardware IP: %v\n", ip)
			t.Log("\n")
		}

		mode := d.Mode(mac)
		if mode != test.mode {
			t.Fatalf("unexpected mode, want: %s, got: %s\n", test.mode, mode)
		}

		if mode == "" {
			continue
		}

		conf := d.NetConfig(mac)
		if conf.Address.String() != test.conf.Address.String() {
			t.Fatalf("unexpected address, want: %s, got: %s\n", test.conf.Address, conf.Address)
		}
		if conf.Netmask.String() != test.conf.Netmask.String() {
			t.Fatalf("unexpected address, want: %s, got: %s\n", test.conf.Netmask, conf.Netmask)
		}
		if conf.Gateway.String() != test.conf.Gateway.String() {
			t.Fatalf("unexpected address, want: %s, got: %s\n", test.conf.Gateway, conf.Gateway)
		}

		osie := d.ServicesVersion.OSIE
		if osie != test.osie {
			t.Fatalf("unexpected osie version, want: %s, got: %s\n", test.osie, osie)
		}
	}
}

var tests = map[string]struct {
	mac  string
	mode string
	conf IP
	osie string
	json string
}{
	"unknown": {
		mac:  "84:b5:9c:cf:17:ff",
		mode: "",
		json: discovered,
	},
	"discovered": {
		mac:  "84:b5:9c:cf:17:01",
		mode: "discovered",
		conf: IP{
			Address: net.ParseIP("10.250.142.74"),
			Gateway: net.ParseIP("10.250.142.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: discovered,
	},
	"management no instance": {
		mac:  "00:25:90:f6:2f:2d",
		mode: "management",
		conf: IP{
			Address: net.ParseIP("10.255.252.16"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: noInstance,
	},
	"management with instance": {
		mac:  "00:25:90:f6:28:5b",
		mode: "management",
		conf: IP{
			Address: net.ParseIP("10.255.252.15"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: withInstance,
	},
	"instance": {
		mac:  "00:25:90:e7:68:da",
		mode: "instance",
		conf: IP{
			Address: net.ParseIP("147.75.193.106"),
			Gateway: net.ParseIP("147.75.193.105"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: withInstance,
	},
	"preinstalling": {
		mac:  "00:25:99:e7:6c:78",
		mode: "hardware",
		conf: IP{
			Address: net.ParseIP("172.16.0.16"),
			Gateway: net.ParseIP("172.16.0.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: preinstalling,
	},
	"deprovisioning": {
		mac:  "fc:15:b4:97:04:e5",
		mode: "instance",
		conf: IP{
			Address: net.ParseIP("172.16.0.14"),
			Gateway: net.ParseIP("172.16.0.13"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: deprovisioning,
	},
	"provisioning": {
		mac:  "fc:15:b4:97:04:f5",
		mode: "instance",
		conf: IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: provisioning,
	},
	"provisioning with custom service": {
		mac:  "fc:15:b4:97:04:f5",
		mode: "instance",
		osie: "v19.01.01.00",
		conf: IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: provisioningWithService,
	},
}

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

	noInstance = `
	{
	  "arch": "x86_64",
	  "bonding_mode": 4,
	  "efi_boot": true,
	  "facility_code": "lab1",
	  "id": "d7e1feaf-d6d5-4d6c-8d16-5c6913be2dea",
	  "instance": {},
	  "management": {
	    "address": "10.255.252.16",
	    "gateway": "10.255.252.1",
	    "netmask": "255.255.255.0"
	  },
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
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "arch": "aarch64",
  "name": "sled3.arm1.d11.lab1.packet.net",
  "type": "sled",
  "state": "preinstalling",
  "vlan_id": "122",
  "efi_boot": true,
  "instance": {},
  "plan_slug": "c1.large.arm",
  "management": {
    "type": "ipmi",
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0"
  },
  "bonding_mode": 4,
  "ip_addresses": [
    {
      "cidr": 30,
      "type": "data",
      "public": false,
      "address": "172.16.0.16",
      "enabled": true,
      "gateway": "172.16.0.15",
      "netmask": "255.255.255.252",
      "network": "172.16.0.14",
      "management": true,
      "address_family": 4
    },
    {
      "type": "ipmi",
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0"
    }
  ],
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "facility_code": "lab1",
  "network_ports": [
    {
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "data": {
        "mac": "00:25:99:e7:6c:78",
        "bond": "bond0"
      },
      "name": "eth0",
      "type": "data",
      "connected_port": {
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/4:2",
        "type": "data"
      }
    },
    {
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "data": {
        "mac": "38:bc:01:c6:cc:de",
        "bond": null
      },
      "name": "eth01",
      "type": "data"
    },
    {
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "data": {
        "mac": "00:25:99:e7:6c:79",
        "bond": "bond0"
      },
      "name": "eth1",
      "type": "data",
      "connected_port": {
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/5:2",
        "type": "data"
      }
    },
    {
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "data": {
        "mac": "fc:15:b4:97:04:e7",
        "bond": null
      },
      "name": "ipmi0",
      "type": "ipmi",
      "connected_port": {
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "Fa0/3",
        "type": "data"
      }
    }
  ],
  "preinstalled_operating_system_version": {
    "distro": "centos",
    "image_tag": null,
    "os_slug": "centos_7",
    "slug": "centos_7-t1.small.x86",
    "version": "7"
  }
}`
	withInstance = `
	{
	  "arch": "x86_64",
	  "bonding_mode": 4,
	  "efi_boot": true,
	  "facility_code": "lab1",
	  "id": "506ad180-8692-480d-b6c2-3ec7f8d719ac",
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
	deprovisioning = `{
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "arch": "aarch64",
  "name": "sled3.arm1.d11.lab1.packet.net",
  "type": "sled",
  "state": "deprovisioning",
  "vlan_id": "122",
  "efi_boot": true,
  "instance": {
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "state": "deprovisioning",
    "rescue": false,
    "hostname": "testing-layer-2",
    "ssh_keys": [],
    "userdata": null,
    "allow_pxe": true,
    "always_pxe": false,
    "ip_addresses": [],
    "ipxe_script_url": null,
    "operating_system_version": {
      "slug": "deprovision",
      "distro": "centos",
      "os_slug": "deprovision",
      "version": "",
      "image_tag": null
    }
  },
  "plan_slug": "c1.large.arm",
  "management": {
    "type": "ipmi",
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0"
  },
  "bonding_mode": 4,
  "ip_addresses": [
    {
      "cidr": 30,
      "type": "data",
      "public": false,
      "address": "172.16.0.14",
      "enabled": true,
      "gateway": "172.16.0.13",
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "management": true,
      "address_family": 4
    },
    {
      "type": "ipmi",
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0"
    }
  ],
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "facility_code": "lab1",
  "network_ports": [
    {
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "data": {
        "mac": "fc:15:b4:97:04:e5",
        "bond": "bond0"
      },
      "name": "eth0",
      "type": "data",
      "connected_port": {
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/4:2",
        "type": "data"
      }
    },
    {
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "data": {
        "mac": "38:bc:01:c6:cc:de",
        "bond": null
      },
      "name": "eth01",
      "type": "data"
    },
    {
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "data": {
        "mac": "fc:15:b4:97:04:e6",
        "bond": "bond0"
      },
      "name": "eth1",
      "type": "data",
      "connected_port": {
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/5:2",
        "type": "data"
      }
    },
    {
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "data": {
        "mac": "fc:15:b4:97:04:e7",
        "bond": null
      },
      "name": "ipmi0",
      "type": "ipmi",
      "connected_port": {
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "Fa0/3",
        "type": "data"
      }
    }
  ],
  "preinstalled_operating_system_version": {}
}`

	provisioning = `{
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "arch": "aarch64",
  "name": "sled3.arm1.d11.lab1.packet.net",
  "type": "sled",
  "state": "deprovisioning",
  "vlan_id": "122",
  "efi_boot": true,
  "instance": {
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "state": "deprovisioning",
    "rescue": false,
    "hostname": "testing-layer-2",
    "ssh_keys": [],
    "userdata": null,
    "allow_pxe": true,
    "always_pxe": false,
    "ip_addresses": [],
    "ipxe_script_url": null,
    "operating_system_version": {
      "slug": "deprovision",
      "distro": "centos",
      "os_slug": "deprovision",
      "version": "",
      "image_tag": null
    },
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
	]
  },
  "plan_slug": "c1.large.arm",
  "management": {
    "type": "ipmi",
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0"
  },
  "bonding_mode": 4,
  "ip_addresses": [
    {
      "cidr": 30,
      "type": "data",
      "public": false,
      "address": "172.16.0.14",
      "enabled": true,
      "gateway": "172.16.0.13",
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "management": true,
      "address_family": 4
    },
    {
      "type": "ipmi",
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0"
    }
  ],
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "facility_code": "lab1",
  "network_ports": [
    {
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "data": {
        "mac": "fc:15:b4:97:04:f5",
        "bond": "bond0"
      },
      "name": "eth0",
      "type": "data",
      "connected_port": {
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/4:2",
        "type": "data"
      }
    },
    {
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "data": {
        "mac": "38:bc:01:c6:cc:de",
        "bond": null
      },
      "name": "eth01",
      "type": "data"
    },
    {
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "data": {
        "mac": "fc:15:b4:97:04:f6",
        "bond": "bond0"
      },
      "name": "eth1",
      "type": "data",
      "connected_port": {
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/5:2",
        "type": "data"
      }
    },
    {
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "data": {
        "mac": "fc:15:b4:97:04:e7",
        "bond": null
      },
      "name": "ipmi0",
      "type": "ipmi",
      "connected_port": {
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "Fa0/3",
        "type": "data"
      }
    }
  ],
  "preinstalled_operating_system_version": {}
}`

	provisioningWithService = `{
  "id": "6300b237-c417-4264-8a0a-58bce33c303f",
  "arch": "aarch64",
  "name": "sled3.arm1.d11.lab1.packet.net",
  "type": "sled",
  "state": "deprovisioning",
  "vlan_id": "122",
  "efi_boot": true,
  "instance": {
    "id": "93068549-726c-4adc-8b0f-b93692cb78ff",
    "state": "deprovisioning",
    "rescue": false,
    "hostname": "testing-layer-2",
    "ssh_keys": [],
    "userdata": null,
    "allow_pxe": true,
    "always_pxe": false,
    "ip_addresses": [],
    "ipxe_script_url": null,
    "operating_system_version": {
      "slug": "deprovision",
      "distro": "centos",
      "os_slug": "deprovision",
      "version": "",
      "image_tag": null
    },
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
  ]
  },
  "plan_slug": "c1.large.arm",
  "management": {
    "type": "ipmi",
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0"
  },
  "bonding_mode": 4,
  "ip_addresses": [
    {
      "cidr": 30,
      "type": "data",
      "public": false,
      "address": "172.16.0.14",
      "enabled": true,
      "gateway": "172.16.0.13",
      "netmask": "255.255.255.252",
      "network": "172.16.0.12",
      "management": true,
      "address_family": 4
    },
    {
      "type": "ipmi",
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0"
    }
  ],
  "manufacturer": {
    "id": "d31118e9-53ab-48ef-a761-5b8811d9a0f5",
    "slug": "foxconn"
  },
  "facility_code": "lab1",
  "network_ports": [
    {
      "id": "fe2d825c-339a-490f-ae23-a336a4f28228",
      "data": {
        "mac": "fc:15:b4:97:04:f5",
        "bond": "bond0"
      },
      "name": "eth0",
      "type": "data",
      "connected_port": {
        "id": "49614525-e949-4e1a-8564-4bfc93bc441a",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/4:2",
        "type": "data"
      }
    },
    {
      "id": "f7957820-43aa-48d1-b902-b0865b73c34d",
      "data": {
        "mac": "38:bc:01:c6:cc:de",
        "bond": null
      },
      "name": "eth01",
      "type": "data"
    },
    {
      "id": "77ecde07-4b32-408e-bbc0-87295c496f8a",
      "data": {
        "mac": "fc:15:b4:97:04:f6",
        "bond": "bond0"
      },
      "name": "eth1",
      "type": "data",
      "connected_port": {
        "id": "1b8a43bf-80ab-440b-af8f-f9416e9b9a2c",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "xe-0/0/5:2",
        "type": "data"
      }
    },
    {
      "id": "2185ee9c-1c1e-4d70-926a-7404eb41b43b",
      "data": {
        "mac": "fc:15:b4:97:04:e7",
        "bond": null
      },
      "name": "ipmi0",
      "type": "ipmi",
      "connected_port": {
        "id": "ee7e96af-2ea0-4f0b-b169-67814ece9800",
        "data": {
          "mac": null,
          "bond": null
        },
        "name": "Fa0/3",
        "type": "data"
      }
    }
  ],
  "preinstalled_operating_system_version": {},
  "services": {
    "osie":"v19.01.01.00"
  }
}`
)
