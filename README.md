# Boots

[![Build Status](https://github.com/tinkerbell/boots/workflows/For%20each%20commit%20and%20PR/badge.svg)](https://github.com/tinkerbell/boots/actions?query=workflow%3A%22For+each+commit+and+PR%22+branch%3Amain)

This service handles DHCP, PXE, tftp, and iPXE for provisions in the Tinkerbell project.
For more information about the Tinkerbell project, see: [tinkerbell.org](https://tinkerbell.org).

## Running Boots

As boots runs a DHCP server, it's often asked if it is safe to run without any network isolation; the answer is yes. While boots does run a DHCP server, it only allocates an IP address when it recognizes the mac address of the requesting device.

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
# run boots
./boots -h

USAGE
  Run Boots server for provisioning

FLAGS
  -backend-file               [backend] enable the DHCP file backend (default "false")
  -backend-file-path          [backend] DHCP file backend hardware file path
  -backend-kube               [backend] enable DHCP kubernetes backend (default "true")
  -backend-kube-api           [backend] the Kubernetes API URL, used for in-cluster client construction. Only applies if DATA_MODEL_VERSION=kubernetes.
  -backend-kube-namespace     [backend] an optional Kubernetes namespace override to query hardware data from.
  -backend-kubeconfig         [backend] the Kubernetes config file location. Only applies if DATA_MODEL_VERSION=kubernetes.
  -dhcp                       [dhcp] enable DHCP server(receiver) (default "true")
  -dhcp-addr                  [dhcp] local IP and port to listen on for DHCP requests (default "0.0.0.0:67")
  -dhcp-http-ipxe-binary-ip   [dhcp] http ipxe binary server IP address to use in DHCP packets (default "http://172.17.0.3:8080/ipxe/")
  -dhcp-http-ipxe-script-url  [dhcp] http ipxe script server URL to use in DHCP packets (default "http://172.17.0.3/auto.ipxe")
  -dhcp-ip-for-packet         [dhcp] ip address to use in DHCP packets (opt 54, etc) (default "172.17.0.3")
  -dhcp-syslog-ip             [dhcp] syslog server IP address to use in DHCP packets (opt 7) (default "172.17.0.3")
  -dhcp-tftp-ip               [dhcp] tftp server IP address to use in DHCP packets (opt 66, etc) (default "172.17.0.3:69")
  -extra-kernel-args          [http] extra set of kernel args (k=v k=v) that are appended to the kernel cmdline iPXE script.
  -http-addr                  [http] local IP and port to listen on for iPXE http script requests (default "172.17.0.3:80")
  -http-ipxe-binary           [http] enable iPXE http binary server(receiver) (default "true")
  -http-ipxe-script           [http] enable iPXE http script server(receiver) (default "true")
  -ipxe-script-patch          [tftp/http] iPXE script fragment to patch into served iPXE binaries served via TFTP or HTTP
  -log-level                  log level (debug, info) (default "info")
  -osie-url                   [http] url where OSIE(Hook) images are located.
  -syslog                     [syslog] enable syslog server(receiver) (default "true")
  -syslog-addr                [syslog] local IP and port to listen on for syslog messages (default "172.17.0.3:514")
  -tftp                       [tftp] enable iPXE tftp binary server(receiver) (default "true")
  -tftp-addr                  [tftp] local IP and port to listen on for iPXE tftp binary requests (default "172.17.0.3:69")
  -tftp-timeout               [tftp] iPXE tftp binary server requests timeout (default "5s")
  -tink-server                [http] ip:port for the Tink server.
  -tink-server-tls            [http] use TLS for Tink server. (default "false")
  -trusted-proxies            [http] comma separated list of trusted proxies
```

You can use NixOS shell, which will have Go and others

`nix-shell`

### Developing with Standalone Mode

The quickest way to get started is `docker-compose up`. This will start boots in
standalone mode using the example JSON in the `test/` directory. It also starts
a client container that runs some tests.

```sh
docker-compose build # build containers
docker-compose up    # start the network & services
# it's fine to hit control-C twice for fast shutdown
docker-compose down  # stop the network & clean up Docker processes
```

Alternatively you can run boots standalone manually. It requires a few
environment variables for configuration.

`test/standalone-hardware.json` should be safe enough for most developers to
use on the command line locally without getting a call from your network
administrator. That said, you might want to contact them before running a DHCP
server on their network. Best to isolate it in Docker or a VM if you're not
sure.

```sh
export DATA_MODEL_VERSION=standalone
export API_CONSUMER_TOKEN=none
export API_AUTH_TOKEN=none
export BOOTS_STANDALONE_JSON=./test/standalone-hardware.json

# to run on your laptop as a regular user
# DHCP won't work but useful for smoke testing and iterating on http/tftp/syslog
./boots \
	-http-addr 127.0.0.1:9000 \
	-syslog-addr 127.0.0.1:9001 \
	-tftp-addr 127.0.0.01:9002 \
	-dhcp-addr 127.0.0.1:9003

# or run it in a container
# NOTE: not sure the NET_ADMIN cap is necessary
docker run -ti --cap-add=NET_ADMIN --volume $(pwd):/boots alpine:3.14
/boots -dhcp-addr 0.0.0.0:67
```
