# Boots

[![Build Status](https://github.com/tinkerbell/boots/workflows/For%20each%20commit%20and%20PR/badge.svg)](https://github.com/tinkerbell/boots/actions?query=workflow%3A%22For+each+commit+and+PR%22+branch%3Amain)

Boots is the network boot service in the [Tinkerbell stack](https://tinkerbell.org). It is comprised of the following services.

- DHCP server
  - host reservations only
  - mac address based lookups
  - netboot options support
  - backend support
    - Kubernetes
    - file based
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

## Running Boots

The DHCP server of Boots serves explicit host reservations only. This means that only hosts that are configured will be served an IP address and network boot details.

## Interoperability with other DHCP servers

It is not recommended, but it is possible for Boots to be run in networks with another DHCP server(s). To get the intended behavior from Boots one of the following must be true.

1. All DHCP servers are configured to serve the same IPAM info as Boots and Boots is the only DHCP server to provide network boot info.

1. All DHCP servers besides Boots are configured to ignore the MAC addresses that Boots is configured to serve.

1. All DHCP servers are configured to serve the same IP address and network boot details as Boots. In this scenario the DHCP functionality of Boots is redundant. It would most likely be recommended to run Boots with the DHCP server functionality disabled (`-dhcp=false`). See the doc on using your existing DHCP service for more details.

### Local Setup

Running the Tests

```bash
# run the tests
make test
```

Build/Run Boots

```bash
# make the binary
make boots
# run Boots
./boots -h

USAGE
  Run Boots server for provisioning

FLAGS
  -log-level                  log level (debug, info) (default "info")
  -backend-file-path          [backend] the hardware yaml file path for the file backend
  -backend-kube-api           [backend] the Kubernetes API URL, used for in-cluster client construction, kube backend only
  -backend-kube-enabled       [backend] enable the kubernetes backend for DHCP and the HTTP iPXE script (default "true")
  -backend-kube-namespace     [backend] an optional Kubernetes namespace override to query hardware data from, kube backend only
  -backend-kubeconfig         [backend] the Kubernetes config file location, kube backend only
  -backend-file-enabled       [backend] enable the file backend for DHCP and the HTTP iPXE script (default "false")
  -dhcp-addr                  [dhcp] local IP and port to listen on for DHCP requests (default "0.0.0.0:67")
  -dhcp-http-ipxe-binary-ip   [dhcp] http ipxe binary server IP address to use in DHCP packets (default "http://172.17.0.2:8080/ipxe/")
  -dhcp-http-ipxe-script-url  [dhcp] http ipxe script server URL to use in DHCP packets (default "http://172.17.0.2/auto.ipxe")
  -dhcp-ip-for-packet         [dhcp] ip address to use in DHCP packets (opt 54, etc) (default "172.17.0.2")
  -dhcp-syslog-ip             [dhcp] syslog server IP address to use in DHCP packets (opt 7) (default "172.17.0.2")
  -dhcp-tftp-ip               [dhcp] tftp server IP address to use in DHCP packets (opt 66, etc) (default "172.17.0.2:69")
  -dhcp-enabled               [dhcp] enable DHCP server (default "true")
  -http-addr                  [http] local IP and port to listen on for iPXE http script requests (default "172.17.0.2:80")
  -tink-server                [http] ip:port for the Tink server
  -http-ipxe-script-enabled   [http] enable iPXE http script server) (default "true")
  -trusted-proxies            [http] comma separated list of trusted proxies
  -extra-kernel-args          [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script
  -osie-url                   [http] url where OSIE(Hook) images are located
  -tink-server-tls            [http] use TLS for Tink server (default "false")
  -http-ipxe-binary-enabled   [http] enable iPXE http binary server (default "true")
  -syslog-enabled             [syslog] enable syslog server(receiver) (default "true")
  -syslog-addr                [syslog] local IP and port to listen on for syslog messages (default "172.17.0.2:514")
  -ipxe-script-patch          [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -tftp-enbled                [tftp] enable iPXE tftp binary server) (default "true")
  -tftp-timeout               [tftp] iPXE tftp binary server requests timeout (default "5s")
  -tftp-addr                  [tftp] local IP and port to listen on for iPXE tftp binary requests (default "172.17.0.2:69")
```

You can use NixOS shell, which will have Go and other dependencies.

`nix-shell`

### Developing using the file backend

The quickest way to get started is `docker-compose up`. This will start Boots using the file backend. This uses the example Yaml file (hardware.yaml) in the `test/` directory. It also starts a client container that runs some tests.

```sh
docker-compose up --build   # build images and start the network & services
# it's fine to hit control-C twice for fast shutdown
docker-compose down  # stop the network & containers
```

Alternatively you can manually run Boots by itself. It requires a few
flags or environment variables for configuration.

`test/hardware.yaml` should be safe enough for most developers to
use on the command line locally without getting a call from your network
administrator. That said, you might want to contact them before running a DHCP
server on their network. Best to isolate it in Docker or a VM if you're not
sure.

```sh
export BOOTS_OSIE_URL=<http url to the OSIE (Operating System Installation Environment) artifacts>
# For more info on the default OSUE (Hook) artifacts, please see https://github.com/tinkerbell/hook
export BOOTS_BACKEND_FILE_ENABLED=true
export BOOTS_BACKEND_FILE_PATH=./test/hardware.yaml
export BOOTS_EXTRA_KERNEL_ARGS="tink_worker_image=quay.io/tinkerbell/tink-worker:latest"

# By default, Boots needs to bind to low ports (67, 69, 80, 514) so it needs root.
sudo -E ./boots

# or run it in a container
# NOTE: not sure the NET_ADMIN cap is necessary
docker run -ti --cap-add=NET_ADMIN --volume $(pwd):/boots alpine:3.14
/boots -dhcp-addr 0.0.0.0:67
```
