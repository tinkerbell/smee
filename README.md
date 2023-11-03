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

## Running Smee

The DHCP server of Smee serves explicit host reservations only. This means that only hosts that are configured will be served an IP address and network boot details.

## Interoperability with other DHCP servers

It is not recommended, but it is possible for Smee to be run in networks with another DHCP server(s). To get the intended behavior from Smee one of the following must be true.

1. All DHCP servers are configured to serve the same IPAM info as Smee and Smee is the only DHCP server to provide network boot info.

1. All DHCP servers besides Smee are configured to ignore the MAC addresses that Smee is configured to serve.

1. All DHCP servers are configured to serve the same IP address and network boot details as Smee. In this scenario the DHCP functionality of Smee is redundant. It would most likely be recommended to run Smee with the DHCP server functionality disabled (`-dhcp=false`). See the doc on using your existing DHCP service for more details.

### Local Setup

Running the Tests

```bash
# run the tests
make test
```

Build/Run Smee

```bash
# make the binary
make smee
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
  -dhcp-http-ipxe-binary-url          [dhcp] HTTP iPXE binaries URL to use in DHCP packets (default "http://172.17.0.2:8080/ipxe/")
  -dhcp-http-ipxe-script-prepend-mac  [dhcp] prepend the hardware MAC address to iPXE script URL base, http://1.2.3.4/auto.ipxe -> http://1.2.3.4/40:15:ff:89:cc:0e/auto.ipxe (default "true")
  -dhcp-http-ipxe-script-url          [dhcp] HTTP iPXE script URL to use in DHCP packets (default "http://172.17.0.2/auto.ipxe")
  -dhcp-iface                         [dhcp] interface to bind to for DHCP requests
  -dhcp-ip-for-packet                 [dhcp] IP address to use in DHCP packets (opt 54, etc) (default "172.17.0.2")
  -dhcp-syslog-ip                     [dhcp] Syslog server IP address to use in DHCP packets (opt 7) (default "172.17.0.2")
  -dhcp-tftp-ip                       [dhcp] TFTP server IP address to use in DHCP packets (opt 66, etc) (default "172.17.0.2:69")
  -extra-kernel-args                  [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script
  -http-addr                          [http] local IP:Port to listen on for iPXE HTTP script requests (default "172.17.0.2:80")
  -http-ipxe-binary-enabled           [http] enable iPXE HTTP binary server (default "true")
  -http-ipxe-script-enabled           [http] enable iPXE HTTP script server (default "true")
  -osie-url                           [http] URL where OSIE (HookOS) images are located
  -tink-server                        [http] IP:Port for the Tink server
  -tink-server-tls                    [http] use TLS for Tink server (default "false")
  -syslog-addr                        [syslog] local IP:Port to listen on for Syslog messages (default "172.17.0.2:514")
  -syslog-enabled                     [syslog] enable Syslog server(receiver) (default "true")
  -ipxe-script-patch                  [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -tftp-addr                          [tftp] local IP:Port to listen on for iPXE TFTP binary requests (default "172.17.0.2:69")
  -tftp-enabled                       [tftp] enable iPXE TFTP binary server) (default "true")
  -tftp-timeout                       [tftp] iPXE TFTP binary server requests timeout (default "5s")
```

You can use NixOS shell, which will have Go and other dependencies.

`nix-shell`

### Developing using the file backend

The quickest way to get started is `docker-compose up`. This will start Smee using the file backend. This uses the example Yaml file (hardware.yaml) in the `test/` directory. It also starts a client container that runs some tests.

```sh
docker compose up --build   # build images and start the network & services
# it's fine to hit control-C twice for fast shutdown
docker compose down  # stop the network & containers
```

Alternatively you can manually run Smee by itself. It requires a few
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
