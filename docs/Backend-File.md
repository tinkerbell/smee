# File Watcher Backend

This document gives an overview of the file watcher backend.
This backend will read in and watch a file on disk for changes.
The data from this file will then be used for serving DHCP requests.

## Why

This backend exists mainly for testing and development.
It allows the DHCP server to be run without having to spin up any additional backend servers, like [Tink](https://github.com/tinkerbell/tink) or [Cacher](https://github.com/packethost/cacher).

## Usage

```bash
# See the file example/main.go for details on how to select and use this backend in code.
go run example/main.go
```

Below is an example of the format used for this file watcher backend.
See this [example.yaml](../backend/file/testdata/example.yaml) for a full working example of the data model.

```yaml
---
08:00:27:29:4E:67:
  ipAddress: "192.168.2.153"
  subnetMask: "255.255.255.0"
  defaultGateway: "192.168.2.1"
  nameServers:
    - "8.8.8.8"
    - "1.1.1.1"
  hostname: "pxe-virtualbox"
  domainName: "example.com"
  broadcastAddress: "192.168.2.255"
  ntpServers:
    - "132.163.96.2"
    - "132.163.96.3"
  leaseTime: 86400
  domainSearch:
    - "example.com"
  netboot:
    allowPxe: true
    ipxeScriptUrl: "https://boot.netboot.xyz"
52:54:00:aa:88:2a:
  ipAddress: "192.168.2.15"
  subnetMask: "255.255.255.0"
  defaultGateway: "192.168.2.1"
  nameServers:
    - "8.8.8.8"
    - "1.1.1.1"
  hostname: "sandbox"
  domainName: "example.com"
  broadcastAddress: "192.168.2.255"
  ntpServers:
    - "132.163.96.2"
    - "132.163.96.3"
  leaseTime: 86400
  domainSearch:
    - "example.com"
  netboot:
    allowPxe: true
    ipxeScriptUrl: "https://boot.netboot.xyz"
```
