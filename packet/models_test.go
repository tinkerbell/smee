package packet

import (
	"encoding/json"
	"net"
	"os"
	"reflect"
	"testing"
)

func TestInterfaces(t *testing.T) {
	var _ Discovery = (*DiscoveryCacher)(nil)
	var _ Discovery = (*DiscoveryTinkerbellV1)(nil)
}

func TestNewDiscoveryCacher(t *testing.T) {
	t.Log("DATA_MODEL_VERSION (should be empty to use cacher):", os.Getenv("DATA_MODEL_VERSION"))

	for name, test := range tests {
		t.Log(name)
		d, err := NewDiscovery([]byte(test.json))
		if err != nil {
			t.Fatal("NewDiscovery", err)
		}
		dc := (*d).(*DiscoveryCacher)
		if dc.PrimaryDataMAC().String() != test.primaryDataMac {
			t.Fatalf("unexpected primary data mac, want: %s, got: %s", test.primaryDataMac, dc.PrimaryDataMAC())
		}
	}
}

func TestNewDiscoveryTinkerbell(t *testing.T) {
	os.Setenv("DATA_MODEL_VERSION", "1")
	t.Log("DATA_MODEL_VERSION:", os.Getenv("DATA_MODEL_VERSION"))

	for name, test := range tinkerbellTests {
		t.Log(name)
		d, err := NewDiscovery([]byte(test.json))
		if err != nil {
			t.Fatal("NewDiscovery", err)
		}
		dt := (*d).(*DiscoveryTinkerbellV1)

		mac, err := net.ParseMAC(test.mac)
		if err != nil {
			t.Fatal("parse test mac", err)
		}

		if dt.Network.InterfaceByMac(mac).DHCP.IP.Address.String() != test.ip.Address.String() {
			t.Fatalf("unexpected ip, want: %v, got: %v", test.ip, dt.Network.InterfaceByMac([]byte(test.mac)).DHCP.IP.Address)
		}
	}
}

func TestNewDiscoveryMismatch(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("TestDiscoveryMismatch should have panicked")
			}
		}()

		os.Setenv("DATA_MODEL_VERSION", "1")
		t.Log("DATA_MODEL_VERSION:", os.Getenv("DATA_MODEL_VERSION"))

		for name, test := range tinkerbellTests {
			t.Log(name)
			d, err := NewDiscovery([]byte(test.json))
			if err != nil {
				t.Fatal("NewDiscovery", err)
			}
			dt := (*d).(*DiscoveryCacher)
			t.Log(dt)
		}
	}()
}

func TestDiscoveryCacher(t *testing.T) {
	for name, test := range tests {
		t.Log(name)
		d := DiscoveryCacher{}

		if err := json.Unmarshal([]byte(test.json), &d); err != nil {
			t.Fatal(test.mode, err)
		}

		mac, err := net.ParseMAC(test.mac)
		if err != nil {
			t.Fatal(test.mode, err)
		}

		// suggest we leave this here for the time being for ease of test why we're not seeing what we expect
		t.Logf("MacType: %s", d.MacType(mac.String()))
		t.Logf("MacIsType=data: %v", d.MacIsType(mac.String(), "data"))
		t.Logf("primaryDataMac: %s", d.PrimaryDataMAC().HardwareAddr().String())
		t.Logf("MAC: %v", d.MAC())
		t.Logf("d.mac: %v", d.mac)
		t.Logf("mac: %s", mac.String())
		if d.Instance() != nil {
			t.Logf("instance: %v", d.Instance())
			t.Logf("instanceId: %s", d.Instance().ID)
			t.Logf("instance State: %s", d.Instance().State)
		}
		t.Logf("hardware IP: %v", d.hardwareIP())
		t.Log("")
		h := *d.Hardware()
		for _, ip := range h.HardwareIPs() {
			t.Logf("hardware IP: %v", ip)
			t.Log("")
		}

		d.SetMAC(mac)
		mode := d.Mode()
		if mode != test.mode {
			t.Fatalf("unexpected mode, want: %s, got: %s", test.mode, mode)
		}

		if mode == "" {
			continue
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

		osie := d.ServicesVersion.Osie
		if osie != test.osie {
			t.Fatalf("unexpected osie version, want: %s, got: %s", test.osie, osie)
		}
	}
}

func TestDiscoveryTinkerbell(t *testing.T) {
	for name, test := range tinkerbellTests {
		t.Log(name)
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
		t.Log("")
		if d.Network.InterfaceByMac(mac).DHCP.IP.Address.String() != test.ip.Address.String() {
			t.Fatalf("unexpected ip, want: %v, got: %v", test.ip, d.Network.InterfaceByMac(mac).DHCP.IP)
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
		t.Logf("netboot ipxe: %v", d.Network.InterfaceByMac(mac).Netboot.IPXE)
		t.Logf("netboot osie: %v", d.Network.InterfaceByMac(mac).Netboot.Osie)
		t.Log("")

		t.Logf("metadata: %v", d.Metadata)
		t.Logf("metadata state: %v", d.Metadata.State)
		t.Logf("metadata bonding_mode: %v", d.Metadata.BondingMode)
		t.Logf("metadata manufacturer: %v", d.Metadata.Manufacturer)
		if d.Instance() != nil {
			t.Logf("instance: %v", d.Instance())
			t.Logf("instance id: %s", d.Instance().ID)
			t.Logf("instance state: %s", d.Instance().State)
		}
		t.Logf("metadata custom: %v", d.Metadata.Custom)
		t.Logf("metadata facility: %v", d.Metadata.Facility)

		//t.Logf("hardware IP: %v", d.hardwareIP())
		t.Log("")
		h := *d.Hardware()
		for _, ip := range h.HardwareIPs() {
			t.Logf("hardware IP: %v", ip)
			t.Log("")
		}

		d.SetMAC(mac)
		mode := d.Mode()
		if mode != test.mode {
			t.Fatalf("unexpected mode, want: %s, got: %s", test.mode, mode)
		}

		if mode == "" {
			continue
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
	}
}

var tinkerbellTests = map[string]struct {
	id            string
	mac           string
	ip            IP
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
	json          string
}{
	"new_structure": {
		id:  "fde7c87c-d154-447e-9fce-7eb7bdec90c0",
		mac: "ec:0d:9a:c0:01:0c",
		ip: IP{
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
		ip: IP{
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
		ip: IP{
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
		json:        fullStructTinkerbell,
	},
}

var tests = map[string]struct {
	mac            string
	primaryDataMac string
	mode           string
	conf           IP
	osie           string
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
		conf: IP{
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
		conf: IP{
			Address: net.ParseIP("10.255.252.16"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: noInstance,
	},
	"management with instance": {
		mac:            "00:25:90:f6:28:5b",
		primaryDataMac: "00:25:90:e7:68:da",
		mode:           "management",
		conf: IP{
			Address: net.ParseIP("10.255.252.15"),
			Gateway: net.ParseIP("10.255.252.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: withInstance,
	},
	"instance": {
		mac:            "00:25:90:e7:68:da",
		primaryDataMac: "00:25:90:e7:68:da",
		mode:           "instance",
		conf: IP{
			Address: net.ParseIP("147.75.193.106"),
			Gateway: net.ParseIP("147.75.193.105"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: withInstance,
	},
	"preinstalling": {
		mac:            "00:25:99:e7:6c:78",
		primaryDataMac: "00:25:99:e7:6c:78",
		mode:           "hardware",
		conf: IP{
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
		conf: IP{
			Address: net.ParseIP("172.16.0.14"),
			Gateway: net.ParseIP("172.16.0.13"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: deprovisioning,
	},
	"provisioning": {
		mac:            "fc:15:b4:97:04:f5",
		primaryDataMac: "fc:15:b4:97:04:f5",
		mode:           "instance",
		conf: IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: provisioning,
	},
	"provisioning with custom service": {
		mac:            "fc:15:b4:97:04:f5",
		primaryDataMac: "fc:15:b4:97:04:f5",
		mode:           "instance",
		osie:           "v19.01.01.00",
		conf: IP{
			Address: net.ParseIP("147.75.14.16"),
			Gateway: net.ParseIP("147.75.14.15"),
			Netmask: net.ParseIP("255.255.255.252"),
		},
		json: provisioningWithService,
	},
	"full structure cacher": {
		mac:            "f4:79:2b:fa:4f:ae",
		primaryDataMac: "e4:45:19:c4:ba:50",
		mode:           "management",
		conf: IP{
			Address: net.ParseIP("10.255.3.13"),
			Gateway: net.ParseIP("10.255.3.1"),
			Netmask: net.ParseIP("255.255.255.0"),
		},
		json: fullStructCacher,
	},
}

const (
	newJsonStructUseDefaults = `
	{
	  "id":"fde7c87c-d154-448e-9fce-7eb7bdec90c0",
	  "network":{
		 "interfaces":[
			{
			   "dhcp":{
				  "mac":"ec:0d:9a:c0:01:0d",
				  "ip":{
					 "address":"192.168.1.5",
					 "netmask":"255.255.255.248",
					 "gateway":"192.168.1.1"
				  },
				  "hostname":"server001",
				  "name_servers": ["1.2.3.4"],
				  "time_servers": [],
				  "arch":"x86_64",
				  "uefi":false
			   },
			   "netboot":{
				  "allow_pxe":true,
				  "allow_workflow":true,
				  "ipxe":{
					 "url":"http://url/menu.ipxe",
					 "contents":"#!ipxe"
				  },
				  "osie":{
					 "kernel":"vmlinuz-x86_64",
					 "initrd":"",
					 "base_url":""
				  }
			   }
			}
		 ]
	  }
}
`

	newJsonStruct = `
	{
	  "id":"fde7c87c-d154-447e-9fce-7eb7bdec90c0",
	  "network":{
		 "interfaces":[
			{
			   "dhcp":{
				  "mac":"ec:0d:9a:c0:01:0c",
				  "ip":{
					 "address":"192.168.1.5",
					 "netmask":"255.255.255.248",
					 "gateway":"192.168.1.1"
				  },
				  "hostname":"server001",
				  "lease_time":172801,
				  "name_servers": [],
				  "time_servers": [],
				  "arch":"x86_64",
				  "uefi":false
			   },
			   "netboot":{
				  "allow_pxe":true,
				  "allow_workflow":true,
				  "ipxe":{
					 "url":"http://url/menu.ipxe",
					 "contents":"#!ipxe"
				  },
				  "osie":{
					 "kernel":"vmlinuz-x86_64",
					 "initrd":"",
					 "base_url":""
				  }
			   }
			}
		 ]
	  },
	  "metadata":{
		 "state":"",
		 "bonding_mode":5,
		 "manufacturer":{
			"id":"",
			"slug":""
		 },
		 "instance":{},
		 "custom":{
			"preinstalled_operating_system_version":{},
			"private_subnets":[]
		 },
		 "facility":{
			"plan_slug":"c2.medium.x86",
			"plan_version_slug":"",
			"facility_code":"ewr1"
		 }
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
		  "operating_system_version": {
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
					"options": ["-L", "ROOT"]
				  },
				  "device": "/dev/sda3",
				  "format": "ext4",
				  "point": "/"
				}
			  },
			  {
				"mount": {
				  "create": {
					"options": ["-L", "SWAP"]
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
	}`
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
	fullStructCacher = `{
  "id": "55639911-2278-498c-b364-8b2a62f5493c",
  "name": "test.dfw2.packet.net",
  "type": "server",
  "state": "in_use",
  "allow_pxe": false,
  "preinstalled_operating_system_version": {},
  "bonding_mode": 4,
  "manufacturer": {
    "id": "d2ea68db-a82a-4273-a590-594cf311ed52",
    "slug": "dell"
  },
  "plan_slug": "n2.xlarge.x86",
  "facility_code": "dfw2",
  "private_subnets": [
    "10.0.0.0/8"
  ],
  "plan_version_slug": "n2.xlarge.x86.01",
  "management": {
    "address": "10.255.3.13",
    "gateway": "10.255.3.1",
    "netmask": "255.255.255.0",
    "type": "ipmi"
  },
  "arch": "x86_64",
  "efi_boot": false,
  "network_ports": [
    {
      "id": "c31fb859-1fe0-4a53-b42b-4cfb45fb7185",
      "type": "data",
      "name": "eth0",
      "data": {
        "mac": "e4:45:19:c4:ba:50",
        "bond": "bond0"
      },
      "connected_ports": [
        {
          "id": "dc76eb4f-6e21-4b6a-b622-08d1970b2224",
          "type": "data",
          "name": "xe-0/0/14:0",
          "data": {
            "mac": null,
            "bond": null
          },
          "hostname": "test.test.dfw2.packet.net"
        }
      ]
    },
    {
      "id": "b676f474-5dc3-4337-8d57-b25b8f7febdf",
      "type": "data",
      "name": "eth1",
      "data": {
        "mac": "e4:45:19:c4:ba:51",
        "bond": "bond1"
      },
      "connected_ports": [
        {
          "id": "a7e87953-0b44-4d4d-af63-2fac95083549",
          "type": "data",
          "name": "xe-0/0/15:0",
          "data": {
            "mac": null,
            "bond": null
          },
          "hostname": "test.test.dfw2.packet.net"
        }
      ]
    },
    {
      "id": "36ba8732-05a4-46c6-857f-8e5765055394",
      "type": "data",
      "name": "eth2",
      "data": {
        "mac": "e4:45:19:c4:ba:52",
        "bond": "bond0"
      },
      "connected_ports": [
        {
          "id": "d10295d3-2bc6-4e76-afdc-e28383ba0766",
          "type": "data",
          "name": "xe-1/0/14:0",
          "data": {
            "mac": null,
            "bond": null
          },
          "hostname": "test.test.dfw2.packet.net"
        }
      ]
    },
    {
      "id": "b36277ed-602d-4736-8fa3-fc1fa647e1d4",
      "type": "data",
      "name": "eth3",
      "data": {
        "mac": "e4:45:19:c4:ba:53",
        "bond": "bond1"
      },
      "connected_ports": [
        {
          "id": "e8e9d7a2-bc7b-4a3d-9512-8f10b5a3632a",
          "type": "data",
          "name": "xe-1/0/15:0",
          "data": {
            "mac": null,
            "bond": null
          },
          "hostname": "test.test.dfw2.packet.net"
        }
      ]
    },
    {
      "id": "2f038e8a-9381-4dc0-8928-3a621a84fd09",
      "type": "ipmi",
      "name": "ipmi0",
      "data": {
        "mac": "f4:79:2b:fa:4f:ae",
        "bond": null
      },
      "connected_ports": [
        {
          "id": "1fdcf403-59b3-40e5-a581-885d1349732e",
          "type": "data",
          "name": "ge-0/0/29",
          "data": {
            "mac": null,
            "bond": null
          },
          "hostname": "rest.test.dfw2.packet.net"
        }
      ]
    }
  ],
  "instance": {
    "id": "331d355a-1925-4ecf-ab6b-75990733c50c",
    "state": "active",
    "hostname": "test",
    "allow_pxe": false,
    "rescue": false,
    "operating_system_version": {
      "slug": "vmware_esxi_7_0",
      "distro": "vmware",
      "version": "7.0",
      "image_tag": "abc123",
      "os_slug": "vmware_esxi_7_0"
    },
    "ipxe_script_url": null,
    "always_pxe": false,
    "userdata": "",
    "storage": {
      "disks": [
        {
          "device": "/dev/sda",
          "wipeTable": true,
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
          ]
        }
      ],
      "filesystems": [
        {
          "mount": {
            "device": "/dev/sda3",
            "format": "ext4",
            "point": "/",
            "create": {
              "options": [
                "-L",
                "ROOT"
              ]
            }
          }
        },
        {
          "mount": {
            "device": "/dev/sda2",
            "format": "swap",
            "point": "none",
            "create": {
              "options": [
                "-L",
                "SWAP"
              ]
            }
          }
        }
      ]
    },
    "ip_addresses": [],
    "ssh_keys": [],
    "tags": [],
    "customdata": {},
    "project": {
      "id": "b9638412-a71e-4390-8366-98bc6244725f",
      "name": "Test",
      "organization": {
        "id": "f27c556e-9476-4703-932f-a198b587c60d",
        "name": "Testing123"
      },
      "primary_owner": {
        "id": "ca8c6279-1d01-4426-83de-cf7f0d62f0ed",
        "full_name": "Tester Jester"
      }
    },
    "network_ready": false,
    "crypted_root_password": "r3d4c73d"
  },
  "vlan_id": null,
  "ip_addresses": [
    {
      "address": "172.16.10.73",
      "netmask": "255.255.255.254",
      "gateway": "172.16.10.72",
      "address_family": 4,
      "public": false,
      "management": true,
      "enabled": true,
      "network": "172.16.10.72",
      "cidr": 31,
      "type": "data",
      "port": "bond0"
    },
    {
      "address": "10.255.3.13",
      "gateway": "10.255.3.1",
      "netmask": "255.255.255.0",
      "type": "ipmi"
    }
  ],
  "services": {}
}`
)
