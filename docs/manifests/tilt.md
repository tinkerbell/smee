# Tilt

This deployment method is for quick local development. Tilt will build and deploy Smee to the Kubernetes cluster pointed to in the current context of your Kubernetes config file. It will use the KinD manifest, documented [here](KIND.md), for deployment.

## Prerequisites

- [Tilt >= v0.28.1](https://docs.tilt.dev/install.html)
- Go >= 1.18
- [Kubectl >= v1.23.4](https://www.downloadkubernetes.com/)
- KinD cluster

## Steps

1. Deploy Smee

   Be sure you have updated `MIRROR_BASE_URL`, `PUBLIC_IP`, `PUBLIC_SYSLOG_FQDN`, and `TINKERBELL_GRPC_AUTHORITY` env variables in the `manifests/kustomize/base/deployment.yaml` file.
   This deployment method uses the kustomize kind overlay (`manifests/kustomize/overlays/kind`). See the `Tiltfile` modify this.

   ```bash
   # Deploy Smee with Tilt
   tilt up --stream
   ```

2. Watch the logs

   ```bash
   kubectl -n tinkerbell logs -f -l app=tinkerbell-smee
   ```
