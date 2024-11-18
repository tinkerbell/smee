local_resource('compile smee',
  cmd='make cmd/smee/smee-linux-amd64',
  deps=["go.mod", "go.sum", "internal", "Dockerfile", "cmd/smee/main.go", "cmd/smee/flag.go", "cmd/smee/backend.go"]
)
docker_build(
    'quay.io/tinkerbell/smee',
    '.',
    dockerfile='Dockerfile',
)
#k8s_yaml(kustomize('./manifests/kustomize/overlays/k3d'))
default_registry('ttl.sh/meohmy-dghentld')

trusted_proxies = os.getenv('trusted_proxies', '')
lb_ip = os.getenv('LB_IP', '192.168.2.114')
stack_version = os.getenv('STACK_CHART_VERSION', '0.5.0')
layer2_interface = os.getenv('LAYER2_INTERFACE', 'eth1')

load('ext://helm_resource', 'helm_resource')
helm_resource('stack',
            chart='oci://ghcr.io/tinkerbell/charts/stack',
            namespace='tink',
            image_deps=['quay.io/tinkerbell/smee'],
            image_keys=[('smee.image')],
            flags=[
              '--create-namespace',
              '--version=%s' % stack_version,
              '--set=global.trustedProxies={%s}' % trusted_proxies,
              '--set=global.publicIP=%s' % lb_ip,
              '--set=stack.kubevip.interface=%s' % layer2_interface,
              '--set=stack.relay.sourceInterface=%s' % layer2_interface,
            ],
            release_name='tink-stack'
)
