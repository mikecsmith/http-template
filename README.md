# http-template

My starting point for new Go HTTP services. Standard library first, no frameworks, opinionated about the small set of things every service needs on day one except auth so I don't rebuild them every time.

If you're reading this and you're not me — feel free to use it. Nothing here is secret and nothing is supported.

## What you get

- `cmd/server` — `main` + `run(ctx, args, getenv, out, sig, ready)` in the [Mat Ryer style][ryer], with `NewServer` assembling the handler graph and a single `addRoutes` registering everything.
- Per-route middleware chain ([Alex Edwards' `slices.Backward` pattern][edwards]): `RequestContext` → `Logging` → `SecureHeaders`.
- `internal/logger` — `slog` with a context handler that pulls request-scoped attrs (`request_id`, `method`, `path`) into every record automatically.
- `internal/request` / `internal/respond` — generic `Decode[T]` / `DecodeValid[T Validator]` and a small JSON response helper with sentinel errors.
- `internal/middleware` — request ID propagation, structured request logging with status-aware levels, baseline security headers (CSP, HSTS, nosniff, etc.).
- `internal/metrics` — OpenTelemetry MeterProvider plumbing wired through `otelhttp` on the server. Ships with the OTel global no-op so it costs nothing until you bolt on a real exporter.
- `internal/config` — flags + env var parsing with a local `flag.FlagSet` (no globals), strict validation, and sensible defaults.
- `cmd/server/main_test.go` — full e2e test that boots `run()` on port `0`, hits the real listener, and shuts it down via context cancel.
- Graceful shutdown via `errgroup` with bounded shutdown contexts for both the HTTP server and the metrics provider.

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

Then:

```sh
curl -i localhost:8080/healthz
curl -i localhost:8080/hello
curl -i -X POST localhost:8080/hello -d '{"name":"mike"}'
```

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

`slog` with a JSON handler. The default level is `error` so probe traffic stays cheap — the `Logging` middleware guards its `slog.Log` call with `Enabled()` so dropped records don't allocate any attrs. Bump `LOG_LEVEL` to `info` while developing.

`RequestContext` injects `request_id`, `method`, and `path` as `slog.Attr` into the request context, and the logger's context handler pulls them into every `slog.*Context` call downstream — so handler code just does `slog.InfoContext(ctx, "...")` and the request fields show up automatically.

## Metrics

`otelhttp.NewHandler` wraps the whole mux in `NewServer`, which means the standard `http.server.*` metrics are produced for every request — including unknown paths and probes — without any per-handler instrumentation. The catch is that they're recorded into OTel's built-in no-op MeterProvider until you opt in.

To turn metrics on, edit `internal/metrics/metrics.go`. The doc comment walks through the shape of the change for the common exporters (Prometheus pull, OTLP push, stdout). Once you've installed a real provider via `otel.SetMeterProvider`, set `METRICS_ENABLED=true` and the existing instrumentation starts emitting.

## Security headers

Conservative baseline for a JSON API: `nosniff`, restrictive CSP, `no-referrer`, two-year HSTS, and `same-origin` CORP. If you serve HTML from the same binary, override CSP for those routes. See `internal/middleware/secure_headers.go` for the rationale on each.

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
