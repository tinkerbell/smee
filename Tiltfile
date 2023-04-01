local_resource(
  'compile boots',
  'make cmd/boots/boots-linux-amd64'
)
docker_build(
    'boots',
    '.',
    dockerfile='Dockerfile',
    only=['./cmd/boots']
)
k8s_yaml(kustomize('./manifests/kustomize/overlays/kind'))
