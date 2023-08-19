# KinD (Kubernetes in Docker)

This describes deploying Smee into a Kubernetes in Docker (KinD) cluster.

## Prerequisites

- [KinD >= v0.12.0](https://kind.sigs.k8s.io/docs/user/quick-start#installation)
- [Kubectl >= v1.23.4](https://www.downloadkubernetes.com/)

## Steps

1. Create KinD cluster

   ```bash
   # Create the KinD cluster
   kind create cluster --config ./manifests/kind/config.yaml
   ```

2. Deploy Smee

   Be sure you have updated `MIRROR_BASE_URL`, `PUBLIC_IP`, `PUBLIC_SYSLOG_FQDN`, and `TINKERBELL_GRPC_AUTHORITY` env variables in the [`manifests/kustomize/base/deployment.yaml`](../../manifests/kustomize/base/deployment.yaml) file.

   ```bash
   # Deploy Smee to KinD
   kubectl kustomize manifests/kustomize/overlays/kind | kubectl apply -f -
   ```

3. Watch the logs

   ```bash
   kubectl -n tinkerbell logs -f -l app=tinkerbell-smee
   ```

> **Note:** KinD will not be able to listen for DHCP broadcast traffic. Using a DHCP relay is recommended.
>
> ```bash
> # Linux direct
> ipaddr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
> sudo -E dhcrelay -id <interface to listen on for DHCP broadcast>  -iu $(ip -o route get ${ipaddr} | cut -d" " -f3) -d ${ipaddr}
>
> # Linux Container
> ipaddr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-control-plane)
> docker run -d --network host --name dhcrelay modem7/dhcprelay:latest -id <interface to listen on for DHCP broadcast>  -iu $(ip -o route get ${ipaddr} | cut -d" " -f3) -d ${ipaddr}
>
> # MacOS TBD
> ```
