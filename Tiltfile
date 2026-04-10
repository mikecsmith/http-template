# Tiltfile — local dev loop for http-template.
#
# Companion files:
#   dev/cluster.yaml  — ctlptl-managed kind cluster + local registry
#   dev/traefik.yaml  — Traefik v3 as a Gateway API controller
#   dev/k8s.yaml      — workload + Gateway + HTTPRoutes (BASE_DOMAIN templated)
#
# Inner loop:
#   1. `mise run cluster-up`  (once per machine)
#   2. `mise run tilt-up`     (or `tilt up`)
#   3. Edit Go files. `go-build` recompiles a fresh linux binary at
#      dist/dev/server, live_update sync's it into the running pod and
#      restart_container() bounces the process — no image rebuild for
#      code changes.
#
# Routing topology (kind extraPortMappings publish node 30080/30443
# out to host 80/443):
#
#   http://*.cluster.localhost            → 301 to https
#   https://cluster.localhost             → http-template
#   https://traefik.cluster.localhost     → Traefik dashboard
#
# TLS is mkcert-issued for *.cluster.localhost + cluster.localhost
# and loaded into the cluster as a Secret named `cluster-tls`. Run
# `mkcert -install` once on a fresh machine so the local CA is trusted.

# ---------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------

# Base domain for everything served through the local gateway. Override
# at invocation time with `tilt up -- --base-domain=foo.localhost`.
config.define_string('base-domain')
cfg = config.parse()
BASE_DOMAIN = cfg.get('base-domain', 'cluster.localhost')

# Guard: only ever run against our local kind cluster.
allow_k8s_contexts('kind-http-template')

default_registry('localhost:5005')

# ---------------------------------------------------------------------
# One-time cluster bootstrap (Gateway CRDs, mkcert certs, TLS secret)
# ---------------------------------------------------------------------

# Gateway API CRDs — pinned, idempotent. Apply via the standard channel
# install manifest published with each release.
local(
    'kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml',
    quiet=True,
)

# mkcert: wildcard cert covering the base domain and any subdomain.
# The presence of the gitignored dev/certs/ directory is the sentinel —
# if you want to regenerate certs (e.g. after `mkcert -uninstall` or
# rotating the local CA), `rm -rf dev/certs` and re-run `tilt up`.
local("""
set -euo pipefail
if [ ! -d dev/certs ]; then
  mkdir -p dev/certs
  mkcert -cert-file dev/certs/cluster.pem -key-file dev/certs/cluster-key.pem \
    "{base}" "*.{base}"
fi
""".format(base=BASE_DOMAIN), quiet=True)

# Load the cert as a TLS secret. `kubectl create --dry-run=client | apply`
# is the standard idempotent-create-or-update pattern.
local("""
kubectl create secret tls cluster-tls \
  --cert=dev/certs/cluster.pem --key=dev/certs/cluster-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -
""", quiet=True)

# ---------------------------------------------------------------------
# Application build + image
# ---------------------------------------------------------------------

# Native Go compile on the host. Faster than building inside Docker;
# the resulting binary gets sync'd into the container by live_update.
local_resource(
    'go-build',
    cmd='mise run dev-build',
    deps=['cmd', 'internal', 'go.mod', 'go.sum'],
)

# Inline Dockerfile mirrors the production image but reads the binary
# from dist/dev/server. Kept inline so the real Dockerfile stays
# goreleaser-shaped ($TARGETPLATFORM layout).
docker_build(
    'http-template',
    context='.',
    dockerfile_contents='''
FROM gcr.io/distroless/static-debian12:nonroot
COPY dist/dev/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
''',
    only=['dist/dev/server'],
    live_update=[
        sync('dist/dev/server', '/server'),
        # distroless has no shell, so run() is unavailable —
        # restart_container is the only way to pick up the new binary.
        restart_container(),
    ],
)

# ---------------------------------------------------------------------
# Manifests
# ---------------------------------------------------------------------

# Traefik install — static, no templating needed.
k8s_yaml('dev/traefik.yaml')

# Workload + Gateway + HTTPRoutes — render BASE_DOMAIN into the file.
k8s_yaml(blob(
    str(read_file('dev/k8s.yaml')).replace('__BASE_DOMAIN__', BASE_DOMAIN),
))

# ---------------------------------------------------------------------
# Resource wiring
# ---------------------------------------------------------------------

k8s_resource(
    'traefik',
    labels=['gateway'],
    # No port_forwards — kind extraPortMappings already publishes
    # NodePort 30080/30443 to host 80/443.
)

k8s_resource(
    'http-template',
    resource_deps=['go-build', 'traefik'],
    labels=['app'],
    links=[
        link('https://{}'.format(BASE_DOMAIN), 'app'),
        link('https://traefik.{}'.format(BASE_DOMAIN), 'traefik dashboard'),
    ],
)
