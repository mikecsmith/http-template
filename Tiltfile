# Tiltfile — local dev loop for http-template.
#
# Pairs with `dev/cluster.yaml` (ctlptl-managed kind cluster + local
# registry on :5005) and `dev/k8s.yaml` (Deployment + Service).
#
# Inner loop:
#   1. `mise run cluster-up` once per machine to spin up the cluster.
#   2. `mise run tilt-up` (or `tilt up`) to start the dev loop.
#   3. Edit Go files. The `go-build` local_resource recompiles a fresh
#      linux binary at dist/dev/server, then live_update sync's it into
#      the running pod and restart_container() bounces the process —
#      no image rebuild required for code changes.
#
# Push target is the ctlptl registry, NOT GHCR — `default_registry`
# rewrites the image name `http-template` to `localhost:5005/http-template`
# so kind pulls locally and we never touch the network.

# Fail loud if Tilt is pointed at a non-local cluster.
allow_k8s_contexts('kind-http-template')

default_registry('localhost:5005')

# Native Go compile on the host. Much faster than building inside Docker
# and the resulting binary gets sync'd into the container by live_update.
local_resource(
    'go-build',
    cmd='mise run dev-build',
    deps=['cmd', 'internal', 'go.mod', 'go.sum'],
)

# Inline Dockerfile mirrors the production image but reads the binary
# from dist/dev/server (single-arch, host-arch). Kept inline so the
# real Dockerfile stays goreleaser-shaped ($TARGETPLATFORM layout).
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
        # distroless has no shell, so run() is unavailable — restart_container
        # is the only way to pick up the new binary.
        restart_container(),
    ],
)

k8s_yaml('dev/k8s.yaml')

k8s_resource(
    'http-template',
    port_forwards=['8080:8080'],
    resource_deps=['go-build'],
)
