# K3D (K3S in Docker)

This describes deploying Boots into a K3S in Docker (K3D) cluster.

## Prerequisites

- [K3D >= v5.4.1](https://k3d.io/v5.4.1/#installation)
- [Kubectl >= v1.23.4](https://www.downloadkubernetes.com/)
- Supported platforms: Linux

### Steps

1. Create K3D cluster

   ```bash
   # Create the K3D cluster
   k3d cluster create --network host --no-lb --k3s-arg "--disable=traefik"
   ```

2. Deploy Boots

   Be sure you have updated `MIRROR_BASE_URL`, `PUBLIC_IP`, `PUBLIC_SYSLOG_FQDN`, and `TINKERBELL_GRPC_AUTHORITY` env variables in the [`manifests/kustomize/base/deployment.yaml`](../../manifests/kustomize/base/deployment.yaml) file.

   ```bash
   # Deploy Boots to K3D
   kubectl kustomize manifests/kustomize/overlays/k3d | kubectl apply -f -
   ```

3. Watch the logs

   ```bash
   kubectl -n tinkerbell logs -f -l app=tinkerbell-boots
   ```
