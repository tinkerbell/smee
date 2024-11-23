# Static IP Address Management in the OSIE ISO

OSIE stands for operating system installation environment. In Tinkerbell we currently have just one, [HookOS](https://github.com/tinkerbell/hook).
Smee has the capability to Patch the HookOS ISO at runtime to include information about the target machine's network configuration. This is enabled by setting the CLI flag `-iso-static-ipam-enabled=true` along with both `-iso-enabled` and `-iso-url`.
This document defines the specification/data format for passing this info to the HookOS ISO.

## Specification/Data format

This is the spec/ data format for passing the static IP address management information to the HookOS ISO.

```ipam=<mac-address>:<vlan-id>:<ip-address>:<netmask>:<gateway>:<hostname>:<dns>:<search-domains>:<ntp>```

Example:

```ipam=de-ad-be-ef-fe-ed:30:192.168.2.193:255.255.255.0:192.168.2.1:server.example.com:1.1.1.1,8.8.8.8:example.com,team.example.com:132.163.97.1,132.163.96.1```

### Fields

Some fields are required so that basic network communication can function properly.

| Field | Description | Required | Example |
|-------|-------------|----------|---------|
| mac-address | MAC address. Must be in dash notation. | Yes |`00-00-00-00-00-00` |
| vlan-id | VLAN ID. Must be a string integer between 0 and 4096 or an empty string for no VLAN tagging. | No | `30` |
| ip-address | IPv4 address. | Yes | `10.148.56.3` |
| netmask | Netmask. | Yes | `255.255.240.0` |
| gateway | IPv4 Gateway. | No | `10.148.56.1` |
| hostname | Hostname for the system. Can be fully qualified or not. | No | `hookos` or `hookos.example.com` |
| dns | Comma separated list of IPv4 DNS nameservers. Must be IPv4 addresses, not hostnames. | Yes | `1.1.1.1,8.8.8.8` |
| search-domains | Comma separated list of search domains. | No | `example.com,example.org` |
| ntp | Comma separated list of IPv4 NTP servers. Must be IPv4 addresses, not hostnames. | No | `132.163.97.1,132.163.96.1` |

## Implementation details

Smee will set the kernel commandline parameter `ipam=` with the above format. In HookOS, there is a service that reads this cmdline parameter and writes the file(s) and runs the command(s) necessary to configure HookOS the use of all the values. See HookOS for more details on the service and how it works.
