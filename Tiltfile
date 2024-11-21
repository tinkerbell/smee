load('ext://restart_process', 'docker_build_with_restart')
load('ext://local_output', 'local_output')
load('ext://helm_resource', 'helm_resource')

local_resource('compile smee',
  cmd='make cmd/smee/smee-linux-amd64',
  deps=["go.mod", "go.sum", "internal", "Dockerfile", "cmd/smee/main.go", "cmd/smee/flag.go", "cmd/smee/backend.go"],
)

#docker_build(
#  'quay.io/tinkerbell/smee',
#  '.',
#  dockerfile='Dockerfile',
#)
docker_build_with_restart(
  'quay.io/tinkerbell/smee',
  '.',
  dockerfile='Dockerfile',
  entrypoint=['/usr/bin/smee'],
  live_update=[
    sync('cmd/smee/smee-linux-amd64', '/usr/bin/smee'),
  ],
)
default_registry('ttl.sh/meohmy-dghentld')

default_trusted_proxies = local_output("kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}' | tr ' ' ','")
trusted_proxies = os.getenv('TRUSTED_PROXIES', default_trusted_proxies)
lb_ip = os.getenv('LB_IP', '')
stack_version = os.getenv('STACK_CHART_VERSION', '0.5.0')
stack_location = os.getenv('STACK_LOCATION', '/home/tink/repos/tinkerbell/charts/tinkerbell/stack') # or a local path like '/home/tink/repos/tinkerbell/charts/tinkerbell/stack'
namespace = 'tink'

if lb_ip == '':
  fail('Please set the LB_IP environment variable. This is required to deploy the stack.')

helm_resource('stack',
  chart=stack_location,
  namespace=namespace,
  image_deps=['quay.io/tinkerbell/smee'],
  image_keys=[('smee.image')],
  flags=[
    '--create-namespace',
    '--version=%s' % stack_version,
    '--set=global.trustedProxies={%s}' % trusted_proxies,
    '--set=global.publicIP=%s' % lb_ip,
    '--set=stack.kubevip.interface=eth1',
    '--set=stack.relay.sourceInterface=eth1',
  ],
  release_name='stack'
)
