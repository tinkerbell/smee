# dhcp

DHCP library with multiple backends. All IP addresses are served as DHCP reservations. There are no lease pools as are normally found in DHCP servers.

## Backends

- [Tink Kubernetes CRDs](https://github.com/tinkerbell/tink/blob/main/config/crd/bases/tinkerbell.org_hardware.yaml)
  - This backend is also the main use case.
    It pulls hardware data from Kubernetes CRDs for use in serving DHCP clients.
- [File based](./docs/Backend-File.md)
  - This backend is for mainly for testing and development.
    It reads a file for hardware data to use in serving DHCP clients.
    See [example.yaml](../backend/file/testdata/example.yaml) for the data model.

## Definitions

**DHCP Reservation:**
A fixed IP address that is reserved for a specific client.

**DHCP Lease:**
An IP address, that can potentially change, that is assigned to a client by the DHCP server.
The IP is typically pulled from a pool or subnet of available IP addresses.
