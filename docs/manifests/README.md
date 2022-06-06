# Deploying Boots

This directory contains the manifests for deploying Boots to various environments. This document will describe how to use the different Boots deployment options.

## Variables

Regardless of the option you choose it is recommended you get started by updating the following environment variables in the [`manifests/kustomize/base/deployment.yaml`](./kustomize/base/deployment.yaml) file to match your setup.

| Variable                    | Description                                                                                         |
| --------------------------- | --------------------------------------------------------------------------------------------------- |
| `TINKERBELL_GRPC_AUTHORITY` | This is the IP:Port that a Tink worker will use for communicated with the Tink server               |
| `MIRROR_BASE_URL`           | The URL from where the "OSIE" or Hook kernel(s) and initrd(s) will be downloaded by netboot clients |
| `PUBLIC_IP`                 | This is the IP that netboot clients and/or DHCP relay's will use to reach Boots                     |
| `PUBLIC_SYSLOG_FQDN`        | This is the IP that syslog clients will use to send messages                                        |

## Deployment Options

- [Kind](kind.md)
- [Kubernetes](kubernetes.md)
- [K3D](k3d.md)
- [Tilt](tilt.md)
