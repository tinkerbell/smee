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
./cmd/boots/boots -h

USAGE
  Run Boots server for provisioning

FLAGS
  -dhcp-addr              IP and port to listen on for DHCP. (default "172.17.0.3:67")
  -http-addr              local IP and port to listen on for the serving iPXE binaries and files via HTTP. (default "172.17.0.3:80")
  -ipxe-enable-http       enable serving iPXE binaries via HTTP. (default "true")
  -ipxe-enable-tftp       enable serving iPXE binaries via TFTP. (default "true")
  -ipxe-remote-http-addr  remote IP and port where iPXE binaries are served via HTTP. Overrides -http-addr for iPXE binaries only.
  -ipxe-remote-tftp-addr  remote IP where iPXE binaries are served via TFTP. Overrides -tftp-addr.
  -ipxe-tftp-addr         local IP and port to listen on for serving iPXE binaries via TFTP (port must be 69). (default "0.0.0.0:69")
  -ipxe-tftp-timeout      local iPXE TFTP server requests timeout. (default "5s")
  -log-level              log level. (default "info")
  -syslog-addr            IP and port to listen on for syslog messages. (default "172.17.0.3:514")
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
./cmd/boots/boots \
	-http-addr 127.0.0.1:9000 \
	-syslog-addr 127.0.0.1:9001 \
	-tftp-addr 127.0.0.01:9002 \
	-dhcp-addr 127.0.0.1:9003

# or run it in a container
# NOTE: not sure the NET_ADMIN cap is necessary
docker run -ti --cap-add=NET_ADMIN --volume $(pwd):/boots alpine:3.14
/boots/cmd/boots -dhcp-addr 0.0.0.0:67
```
