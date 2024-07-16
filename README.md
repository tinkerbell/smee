# Smee

[![Build Status](https://github.com/tinkerbell/smee/workflows/For%20each%20commit%20and%20PR/badge.svg)](https://github.com/tinkerbell/smee/actions?query=workflow%3A%22For+each+commit+and+PR%22+branch%3Amain)

Smee is the network boot service in the [Tinkerbell stack](https://tinkerbell.org), formerly known as `Boots`. It is comprised of the following services.

- DHCP server
  - host reservations only
  - mac address based lookups
  - netboot options support
  - backend support
    - Kubernetes
    - file based
  - ProxyDHCP support
- TFTP server
  - serving iPXE binaries
- HTTP server
  - serving iPXE binaries and iPXE scripts
  - iPXE script serving uses IP authentication
  - backend support
    - Kubernetes
    - file based
- Syslog server
  - receives syslog messages and logs them

## Definitions

**DHCP Reservation:**
A fixed IP address that is reserved for a specific client.

**DHCP Lease:**
An IP address, that can potentially change, that is assigned to a client by the DHCP server.
The IP is typically pulled from a pool or subnet of available IP addresses.

**ProxyDHCP:**
"[A] Proxy DHCP server behaves much like a DHCP server by listening for ordinary DHCP client traffic and responding to certain client requests. However, unlike the DHCP server, the PXE Proxy DHCP server does not administer network addresses, and it only responds to clients that identify themselves as PXE clients.
The responses given by the PXE Proxy DHCP server contain the mechanism by which the client locates the boot servers or the network addresses and descriptions of the supported, compatible boot servers."
-- [IBM](https://www.ibm.com/docs/en/aix/7.1?topic=protocol-preboot-execution-environment-proxy-dhcp-daemon)

## Running Smee

### DHCP Modes

Smee's DHCP functionality can operate in one of the following modes:

1. **DHCP Reservation**  
   To enable this mode set `-dhcp-mode=reservation`.
   Smee will respond to DHCP requests from clients and provide them with IP and next boot info when netbooting. This is the default mode. IP info is all reservation based. There must be a corresponding Hardware record for the requesting client's MAC address.  

1. **Proxy DHCP**  
   To enable this mode set `-dhcp-mode=proxy`.
   Smee will respond to PXE enabled DHCP requests from clients and provide them with next boot info when netbooting. In this mode an existing DHCP server that does not serve network boot information is required. Smee will respond to PXE enabled DHCP requests and provide the client with the next boot info. There must be a corresponding Hardware record for the requesting client's MAC address. The `auto.ipxe` script will be served with the MAC address in the URL and the MAC address will be used to lookup the corresponding Hardware record. Layer 2 access to machines or a DHCP relay agent that will forward the DHCP requests to Smee is required.

1. **Auto Proxy DHCP**
   To enable this mode set `-dhcp-mode=auto-proxy`.
   Smee will respond to PXE enabled DHCP requests from clients and provide them with next boot info when netbooting. In this mode an existing DHCP server that does not serve network boot information is required. In this mode, if no corresponding Hardware record is found for the requesting client's MAC address, Smee will provide the client with a statically defined iPXE script. If a Hardware record is found, then the normal `auto.ipxe` script will be served. There is a builtin static iPXE script. To override the builtin static iPXE script use `--dhcp-http-ipxe-script-url`. If `--dhcp-http-ipxe-script-url` is specified to a non default location (i.e. not Smee) then be sure to also set `--dhcp-http-ipxe-script-prepend-mac=false`. Layer 2 access to machines or a DHCP relay agent that will forward the DHCP requests to Smee is required.

   - When using Smee's auto.ipxe, you'll generally want to set the following flags:  
     - `-dhcp-mode=auto-proxy`
     - `-osie-url <URL to HookOS kernel and initrd>`
     - `-tink-server <IP and port of Tink server>`
     - `-extra-kernel-args="tink_worker_image=quay.io/tinkerbell/tink-worker:<use a version/commit tag>"`
   - When not using Smee's auto.ipxe, you'll generally want to set the following flags:  
     - `-dhcp-mode=auto-proxy`
     - `-dhcp-http-ipxe-script-url=https://boot.netboot.xyz`
     - `-dhcp-http-ipxe-script-prepend-mac=false`

1. **DHCP disabled**  
   To enable this mode set `-dhcp-enabled=false`.
   Smee will not respond to DHCP requests from clients. This is useful when the network has an existing DHCP server that will provide both IP and next boot info and Smee's TFTP and HTTP functionality will be used. The IP address in the Hardware record must be the same as the IP address of the client requesting the `auto.ipxe` script. See this [doc](docs/DHCP.md) for more details. In most situations`--dhcp-http-ipxe-script-prepend-mac=false` should also be set when in this mode.

### Interoperability with other DHCP servers

When a DHCP server exists on the network, Smee should be set to run `proxy` or `auto-proxy` mode. This will allow Smee to provide the next boot information to clients that request it. The existing DHCP server will provide the IP address and other network boot details. Layer 2 access to machines or a DHCP relay agent that will forward the DHCP requests to Smee is required.

It is not recommended, but it is possible for Smee to be run in `reservation` mode in networks with another DHCP server(s). To get the intended behavior from Smee one of the following must be true.

1. All DHCP servers besides Smee are configured to ignore the MAC addresses that Smee is configured to serve.

1. All DHCP servers are configured to serve the same IP address and network boot details as Smee. In this scenario the DHCP functionality of Smee is redundant. It would be recommended to run Smee with the DHCP server functionality disabled (`-dhcp=false`). See the [doc](./docs/DHCP.md) on using your existing DHCP service for more details.

### Local Setup

Running the Tests

```bash
# run the tests
make test
```

Build/Run Smee

```bash
# make the binary
make build
# run Smee
./smee -h

Smee is the DHCP and Network boot service for use in the Tinkerbell stack.

USAGE
  smee [flags]

FLAGS
  -log-level                          log level (debug, info) (default "info")
  -backend-file-enabled               [backend] enable the file backend for DHCP and the HTTP iPXE script (default "false")
  -backend-file-path                  [backend] the hardware yaml file path for the file backend
  -backend-kube-api                   [backend] the Kubernetes API URL, used for in-cluster client construction, kube backend only
  -backend-kube-config                [backend] the Kubernetes config file location, kube backend only
  -backend-kube-enabled               [backend] enable the kubernetes backend for DHCP and the HTTP iPXE script (default "true")
  -backend-kube-namespace             [backend] an optional Kubernetes namespace override to query hardware data from, kube backend only
  -dhcp-addr                          [dhcp] local IP:Port to listen on for DHCP requests (default "0.0.0.0:67")
  -dhcp-enabled                       [dhcp] enable DHCP server (default "true")
  -dhcp-http-ipxe-binary-host         [dhcp] HTTP iPXE binaries host or IP to use in DHCP packets (default "172.17.0.3")
  -dhcp-http-ipxe-binary-path         [dhcp] HTTP iPXE binaries path to use in DHCP packets (default "/ipxe/")
  -dhcp-http-ipxe-binary-port         [dhcp] HTTP iPXE binaries port to use in DHCP packets (default "8080")
  -dhcp-http-ipxe-binary-scheme       [dhcp] HTTP iPXE binaries scheme to use in DHCP packets (default "http")
  -dhcp-http-ipxe-script-host         [dhcp] HTTP iPXE script host or IP to use in DHCP packets (default "172.17.0.3")
  -dhcp-http-ipxe-script-path         [dhcp] HTTP iPXE script path to use in DHCP packets (default "/auto.ipxe")
  -dhcp-http-ipxe-script-port         [dhcp] HTTP iPXE script port to use in DHCP packets (default "8080")
  -dhcp-http-ipxe-script-prepend-mac  [dhcp] prepend the hardware MAC address to iPXE script URL base, http://1.2.3.4/auto.ipxe -> http://1.2.3.4/40:15:ff:89:cc:0e/auto.ipxe (default "true")
  -dhcp-http-ipxe-script-scheme       [dhcp] HTTP iPXE script scheme to use in DHCP packets (default "http")
  -dhcp-iface                         [dhcp] interface to bind to for DHCP requests
  -dhcp-ip-for-packet                 [dhcp] IP address to use in DHCP packets (opt 54, etc) (default "172.17.0.3")
  -dhcp-mode                          [dhcp] DHCP mode (reservation, proxy) (default "reservation")
  -dhcp-syslog-ip                     [dhcp] Syslog server IP address to use in DHCP packets (opt 7) (default "172.17.0.3")
  -dhcp-tftp-ip                       [dhcp] TFTP server IP address to use in DHCP packets (opt 66, etc) (default "172.17.0.3")
  -dhcp-tftp-port                     [dhcp] TFTP server port to use in DHCP packets (opt 66, etc) (default "69")
  -disable-discover-trusted-proxies   [http] disable discovery of trusted proxies from Kubernetes, only available for the Kubernetes backend (default "false")
  -extra-kernel-args                  [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script
  -http-addr                          [http] local IP to listen on for iPXE HTTP script requests (default "172.17.0.3")
  -http-ipxe-binary-enabled           [http] enable iPXE HTTP binary server (default "true")
  -http-ipxe-script-enabled           [http] enable iPXE HTTP script server (default "true")
  -http-port                          [http] local port to listen on for iPXE HTTP script requests (default "8080")
  -ipxe-script-retries                [http] number of retries to attempt when fetching kernel and initrd files in the iPXE script (default "0")
  -ipxe-script-retry-delay            [http] delay (in seconds) between retries when fetching kernel and initrd files in the iPXE script (default "2")
  -osie-url                           [http] URL where OSIE (HookOS) images are located
  -tink-server                        [http] IP:Port for the Tink server
  -tink-server-tls                    [http] use TLS for Tink server (default "false")
  -trusted-proxies                    [http] comma separated list of trusted proxies in CIDR notation
  -otel-endpoint                      [otel] OpenTelemetry collector endpoint
  -otel-insecure                      [otel] OpenTelemetry collector insecure (default "true")
  -syslog-addr                        [syslog] local IP to listen on for Syslog messages (default "172.17.0.3")
  -syslog-enabled                     [syslog] enable Syslog server(receiver) (default "true")
  -syslog-port                        [syslog] local port to listen on for Syslog messages (default "514")
  -ipxe-script-patch                  [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -tftp-addr                          [tftp] local IP to listen on for iPXE TFTP binary requests (default "172.17.0.3")
  -tftp-block-size                    [tftp] TFTP block size a value between 512 (the default block size for TFTP) and 65456 (the max size a UDP packet payload can be) (default "512")
  -tftp-enabled                       [tftp] enable iPXE TFTP binary server) (default "true")
  -tftp-port                          [tftp] local port to listen on for iPXE TFTP binary requests (default "69")
  -tftp-timeout                       [tftp] iPXE TFTP binary server requests timeout (default "5s")
```

### Developing using the file backend

The quickest way to get started is `docker-compose up`. This will start Smee using the file backend. This uses the example Yaml file (hardware.yaml) in the `test/` directory. It also starts a client container that runs some tests.

```sh
docker compose up --build   # build images and start the network & services
# it's fine to hit control-C twice for fast shutdown
docker compose down  # stop the network & containers
```

Alternatively Smee can be run by itself. It requires a few
flags or environment variables for configuration.

`test/hardware.yaml` should be safe enough for most developers to
use on the command line locally without getting a call from your network
administrator. That said, you might want to contact them before running a DHCP
server on their network. Best to isolate it in Docker or a VM if you're not
sure.

```sh
# build the binary
make build

export SMEE_OSIE_URL=<http url to the OSIE (Operating System Installation Environment) artifacts>
# For more info on the default OSIE (Hook) artifacts, please see https://github.com/tinkerbell/hook
export SMEE_BACKEND_KUBE_ENABLED=false
export SMEE_BACKEND_FILE_ENABLED=true
export SMEE_BACKEND_FILE_PATH=./test/hardware.yaml
export SMEE_EXTRA_KERNEL_ARGS="tink_worker_image=quay.io/tinkerbell/tink-worker:latest"

# By default, Smee needs to bind to low ports (67, 69, 514) so it needs root.
sudo -E ./cmd/smee/smee

# clean up the environment variables
unset SMEE_OSIE_URL
unset SMEE_BACKEND_KUBE_ENABLED
unset SMEE_BACKEND_FILE_ENABLED
unset SMEE_BACKEND_FILE_PATH
unset SMEE_EXTRA_KERNEL_ARGS
```
