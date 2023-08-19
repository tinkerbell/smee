local_resource(
  'compile smee',
  'make cmd/smee/smee-linux-amd64'
)
docker_build(
    'quay.io/tinkerbell/smee',
    '.',
    dockerfile='Dockerfile',
    only=['.']
)
k8s_yaml(kustomize('./manifests/kustomize/overlays/kind'))
