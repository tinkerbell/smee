local_resource(
  'compile boots',
  'make boots-linux-amd64'
)
docker_build(
    'quay.io/tinkerbell/boots',
    '.',
    dockerfile='Dockerfile',
    only=['.']
)
k8s_yaml(kustomize('./manifests/kustomize/overlays/kind'))
