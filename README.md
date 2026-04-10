# http-template

My starting point for new Go HTTP services. Standard library first, no frameworks, opinionated about the small set of things every service needs on day one except auth so I don't rebuild them every time.

If you're reading this and you're not me — feel free to use it. Nothing here is secret and nothing is supported.

## Use it as a template

[`gonew`][gonew] copies the template into a new module and rewrites import paths in one shot:

```sh
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/mikecsmith/http-template github.com/you/your-service
cd your-service
go mod tidy
go test ./...
```

The OTel service name defaults to `filepath.Base(os.Args[0])`, so the binary name becomes the service name automatically — no constants to edit after `gonew`.

## Run it

```sh
go run ./cmd/server
# or with overrides
go run ./cmd/server --port 9000 --log-level info --metrics-enabled
PORT=9000 LOG_LEVEL=debug go run ./cmd/server
```

Or build the container locally with goreleaser (matches what CI/release does):

```sh
goreleaser release --snapshot --clean --skip=publish
docker run --rm -p 8080:8080 ghcr.io/mikecsmith/http-template:<snapshot-tag>-amd64
```

Then:

```sh
curl -i localhost:8080/healthz
curl -i localhost:8080/hello
curl -i -X POST localhost:8080/hello -d '{"name":"mike"}'
```

## Local Kubernetes dev loop

For working against a real Gateway API + TLS setup the repo ships a [Tilt][tilt] config that drives a local [kind][kind] cluster managed by [ctlptl][ctlptl], with [Traefik v3][traefik] as the Gateway API controller and [mkcert][mkcert] providing a locally-trusted wildcard cert. Tool versions are pinned in `.mise.toml`:

```sh
mise install
```

One-time per machine (re-run after `rm -rf dev/certs` to rotate the wildcard cert or after removing the Kind cluster):

```sh
mise run cluster-up
```

That installs the mkcert root CA, generates `*.cluster.localhost`, brings up the kind cluster + a local registry on `localhost:5005`, mounts the mkcert CA into the node's trust store, and publishes node ports `30080`/`30443` to host `:80`/`:443`.

Then:

```sh
tilt up
```

Routing topology, all served by Traefik on the kind node:

| URL                                            | Routes to                                |
| ---------------------------------------------- | ---------------------------------------- |
| `http://*.cluster.localhost`                   | 301 → https                              |
| `https://api.cluster.localhost/<app-name>/*`   | `<app-name>` service (prefix stripped)   |
| `https://traefik.cluster.localhost`            | Traefik dashboard                        |

The `api.` host is the shared multi-API entry point: the backend mounts under `api.cluster.localhost/<service-name>/*` and the gateway's `URLRewrite` filter strips the `/<service-name>` prefix before forwarding, so each service keeps a flat route table internally. For this template that means `https://api.cluster.localhost/http-template/hello` reaches the Go handler registered at `GET /hello`. Adding a new service is one HTTPRoute with a fresh prefix — no extra hostnames, no wildcard cert reissuance and matches common enterprise gateway patterns.

Inner loop is binary-only: a `local_resource` cross-compiles `dist/dev/server` on every Go file change, Tilt's `live_update` syncs it into the running pod, and the `tilt-restart-wrapper` re-execs the process — no image rebuild on code changes. The dev image uses `alpine` (rather than the production distroless base) because the restart wrapper needs `touch`/`chmod` at build time and the manifest drops `readOnlyRootFilesystem` so live_update can overwrite `/server`. Production hardening still applies via the goreleaser-built distroless image.

Tear down:

```sh
mise run cluster-down
```

`dev/cluster.yaml` (ctlptl), `dev/traefik.yaml` (Traefik install + GatewayClass), and `dev/k8s.yaml` (workload + Gateway + HTTPRoutes) are the three manifests Tilt drives. The workload manifest uses envsubst-style placeholders (`${BASE_DOMAIN}`, `${APP_NAME}`) that the Tiltfile renders in-memory before applying — `${APP_NAME}` is parsed from `go.mod` so derived projects work without edits, and you can override the base domain via `tilt up -- --base-domain=foo.localhost`.

## Configuration

Every option is settable via flag or env var. Env wins over flag.

| Flag                  | Env                 | Default     | Notes                                    |
| --------------------- | ------------------- | ----------- | ---------------------------------------- |
| `--host`              | `HOST`              | `""`        | Bind host                                |
| `--port`              | `PORT`              | `8080`      | Bind port                                |
| `--log-level`         | `LOG_LEVEL`         | `error`     | `debug` / `info` / `warn` / `error`      |
| `--metrics-enabled`   | `METRICS_ENABLED`   | `false`     | Install a real MeterProvider (see below) |
| `--otel-service-name` | `OTEL_SERVICE_NAME` | binary name | OTel `service.name`                      |
| `--request-timeout`   | `REQUEST_TIMEOUT`   | `5s`        | `http.Server.ReadTimeout`                |
| `--write-timeout`     | `WRITE_TIMEOUT`     | `10s`       | `http.Server.WriteTimeout`               |
| `--idle-timeout`      | `IDLE_TIMEOUT`      | `120s`      | `http.Server.IdleTimeout`                |
| `--shutdown-timeout`  | `SHUTDOWN_TIMEOUT`  | `10s`       | Bound on graceful shutdown               |

## Logging

`slog` with a JSON handler. The default level is `error` so probe traffic stays cheap — the `Logging` middleware guards its `slog.Log` call with `Enabled()` so dropped records don't allocate any attrs. Bump `LOG_LEVEL` to `info` while developing, or `debug` if you need probe logging.

`RequestContext` injects `request_id`, `method`, and `path` as `slog.Attr` into the request context, and the logger's context handler pulls them into every `slog.*Context` call downstream — so handler code just does `slog.InfoContext(ctx, "...")` and the request fields show up automatically.

## Metrics

`otelhttp.NewHandler` wraps the whole mux in `NewServer`, which means the standard `http.server.*` metrics are produced for every request — including unknown paths and probes — without any per-handler instrumentation. The catch is that they're recorded into OTel's built-in no-op MeterProvider until you opt in.

To turn metrics on, edit `internal/metrics/metrics.go`. The doc comment walks through the shape of the change for the common exporters (Prometheus pull, OTLP push, stdout). Once you've installed a real provider via `otel.SetMeterProvider`, set `METRICS_ENABLED=true` and the existing instrumentation starts emitting.

## Security headers

Conservative baseline for a JSON API: `nosniff`, restrictive CSP, `no-referrer`, two-year HSTS, and `same-origin` CORP. If you serve HTML from the same binary, override CSP for those routes. See `internal/middleware/secure_headers.go` for the rationale on each.

## CI, releases & containers

Three GitHub Actions workflows drive the repo:

- **`ci.yaml`** runs on every push and PR: `go test -race`, `golangci-lint`, `govulncheck`, plus a `goreleaser release --snapshot --clean --skip=publish` smoke test that exercises the full multi-arch build + Dockerfile path without pushing. Catches Dockerfile and goreleaser regressions on the PR rather than at release time.
- **`release-please.yaml`** opens or updates a release PR on `main` based on conventional commits. Merging that PR creates the `vX.Y.Z` tag and triggers the goreleaser job in the same workflow, which logs in to GHCR, builds per-arch binaries + images, and stitches them into a multi-arch manifest pushed to `ghcr.io/<owner>/<repo>:X.Y.Z` and `:latest`.
- **`dependabot-automerge.yaml`** auto-merges Dependabot patch and minor bumps across `gomod`, `github-actions`, and `docker` ecosystems once CI is green. Major bumps stay open for review. Requires "Allow auto-merge" and a branch protection rule on `main` requiring CI to pass — see the `gonew` checklist above.

Release flow end to end:

1. Land conventional commits on `main` (`feat:`, `fix:`, `chore:`, …).
2. release-please opens/updates a release PR with the generated changelog and version bump.
3. Merging the release PR creates the `vX.Y.Z` tag, which triggers goreleaser.
4. goreleaser builds `linux/amd64` + `linux/arm64` binaries, builds per-arch images from `Dockerfile`, and stitches them into a multi-arch manifest pushed to GHCR.

Build version, commit, and date are stamped into the binary via `-ldflags` and logged on startup. Local `go build` falls back to Go's embedded `runtime/debug.BuildInfo` (`vcs.revision` truncated to 8 chars, `vcs.time` as the date), so even dev builds report a real commit.

You can run the workflows locally against Docker via [act][act]:

```sh
mise run act-ci
mise run act-dependabot
```

## Layout

```
cmd/server/
  main.go          // main + run() with OS deps injected
  server.go        // NewServer: mux + addRoutes + otelhttp wrapper
  routes.go        // addRoutes + middleware chain
  main_test.go     // end-to-end test of run()
internal/
  config/          // flag + env parsing, no globals
  handle/          // HTTP handlers (healthz, readyz, hello, not_found)
  logger/          // slog context handler
  metrics/         // OTel MeterProvider plumbing (no-op by default)
  middleware/      // request_context, logging, secure_headers
  request/         // generic Decode + DecodeValid
  respond/         // JSON response helpers + sentinel errors
```

## Tests

```sh
go test -race ./...
```

The `cmd/server` test exercises the full lifecycle: it calls `run()` on port `0`, waits for the listener address via the `ready` callback, fires real HTTP requests against the running server, then cancels the context to trigger graceful shutdown. Adding a new endpoint usually means a unit test next to the handler plus one row in the e2e table.

[ryer]: https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/
[edwards]: https://www.alexedwards.net/blog/organize-your-go-middleware-without-dependencies
[gonew]: https://go.dev/blog/gonew
[release-please]: https://github.com/googleapis/release-please
[tilt]: https://tilt.dev
[kind]: https://kind.sigs.k8s.io
[ctlptl]: https://github.com/tilt-dev/ctlptl
[traefik]: https://traefik.io/traefik/
[mkcert]: https://github.com/FiloSottile/mkcert
[mise]: https://mise.jdx.dev
[act]: https://github.com/nektos/act
