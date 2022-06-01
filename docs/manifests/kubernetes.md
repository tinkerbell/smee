# Kubernetes

This deployment requires a running Kubernetes cluster. It can be a single node cluster. It is required to be running directly on a Linux machine, not in a container. This deployment is under development and is not guaranteed to work at this time.

## Prerequisites

TBD

## Steps

1. Deploy Boots

   Be sure you have updated `MIRROR_BASE_URL`, `PUBLIC_IP`, `PUBLIC_SYSLOG_FQDN`, and `TINKERBELL_GRPC_AUTHORITY` env variables in the [`manifests/kustomize/base/deployment.yaml`](../../manifests/kustomize/base/deployment.yaml) file.

   ```bash
   # Deploy Boots to Kubernetes
   kubectl kustomize manifests/kustomize/overlays/dev | kubectl apply -f -
   ```

2. Watch the logs

   ```bash
   kubectl -n tinkerbell logs -f -l app=tinkerbell-boots
   ```
