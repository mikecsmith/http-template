# httplab

A learning project for building idiomatic HTTP services in Go using the standard library only. This project deliberately avoids frameworks like Gin or Fiber — every decision is made in terms of what the Go standard library provides and why.

---

## Project Structure

```
TODO: Add me when done
```

`cmd/` contains binary entrypoints — code that produces an executable. `internal/` contains shared application logic that can be imported by any binary in this module but is invisible to external modules. This boundary is enforced by the Go compiler, not convention.

---

## Design Philosophy

### Standard library only

Go's standard library is unusually capable for HTTP services. `net/http` provides routing, middleware composition, TLS, and HTTP/2. `encoding/json` handles serialisation. `log/slog` (Go 1.21+) provides structured logging. Adding a framework at this stage would obscure how these pieces fit together.

### `internal/` means shared, not pure

`internal/` contains code that can be imported by any binary in this modulebut is invisible to external modules. This is a compiler-enforced visibility boundary, not a statement about purity.

Currently `internal/` contains HTTP boundary code — `respond` writes HTTP responses, `handle` contains request handlers. Both involve I/O. As the project grows, pure business logic functions will also live in `internal/`, called by handlers but independent of HTTP concerns. At that point the functional core/imperative shell distinction becomes meaningful in practice.

### Explicit over implicit

Dependencies are passed as function arguments rather than stored in global state or hidden in structs. Every function's requirements are visible at the call site.

---

## The `run` Pattern

Go's `main` function cannot return an error and cannot be tested directly. The `run` pattern solves this by moving all real logic into a `run` function that `main` simply calls:

```go
func main() {
    ctx := context.Background()
    if err := run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
}
```

`main` has exactly one job: call `run`, translate any error into a non-zero exit code, and write it to stderr. Everything else — server setup, route registration, graceful shutdown — lives in `run`. This makes the entire application startup testable by calling `run` directly in tests with controlled inputs.

`run` returns an `error` rather than an exit code. Go's `error` type is an interface that carries a description of what went wrong — far more information than an integer exit code, and composable via wrapping with `fmt.Errorf("context: %w", err)`.

---

## HTTP Routing with `http.NewServeMux`

`http.NewServeMux()` creates a request multiplexer — a router that maps URL patterns to handler functions. Since Go 1.22, the mux supports method-prefixed patterns:

```go
mux.Handle("GET /hello", handle.Hello())
```

The `GET ` prefix means only `GET` requests match this route. Without it, all HTTP methods match. This is part of the standard library — no third-party router needed for the vast majority of use cases.

Routes are registered in a dedicated `addRoutes` function rather than inline in `run`. This keeps route registration explicit, readable, and visible in one place.

---

## Handlers

A Go HTTP handler is any value that implements the `http.Handler` interface:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

`http.HandlerFunc` is a function type that satisfies this interface, meaning any function with the signature `func(http.ResponseWriter, *http.Request)` can be used as a handler directly.

Our handlers use a factory pattern — they return an `http.HandlerFunc` rather than being one:

```go
func Hello() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // handle the request
    }
}
```

The factory pattern earns its place when a handler needs dependencies — a database connection, a config value. Those are passed into the factory and captured by the returned closure, rather than being stored in global state:

```go
func Hello(db *sql.DB, cfg Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // db and cfg are available here via closure
    }
}
```

---

## JSON Encoding

Go's `encoding/json` package serialises Go values to JSON. Struct tags control field names and behaviour in the output:

```go
type ResponseError struct {
    Code    int    `json:"-"`                   // excluded from JSON entirely
    Message string `json:"error"`               // appears as "error" in JSON
    Details string `json:"details,omitempty"`   // omitted when empty string
}
```

Tags are backtick-delimited string literals attached to struct fields. The `json` key tells the encoder what to do with that field. `-` means skip it. `omitempty` means skip it if it holds the zero value for its type (empty string, 0, nil, etc.).

### Why `json.Marshal` rather than `json.NewEncoder`

`json.NewEncoder(w).Encode(data)` writes directly to the `http.ResponseWriter` — efficient, but once you start writing you cannot change the HTTP status code. If encoding fails mid-stream, the client receives a truncated response with an incorrect 200 header.

`json.Marshal` encodes to a `[]byte` in memory first. If it fails, no bytes have been written and we can still write a correct 500 response. Correctness beats the marginal allocation saving for a response helper:

```go
b, err := json.Marshal(data)
if err != nil {
    // nothing written yet — we can still respond with 500
    WithError(w, r, ErrInternalServerError)
    return
}
// only now do we commit to the status code and write
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
w.Write(b)
```

---

## The `respond` Package

Rather than each handler independently encoding JSON and setting headers, `respond` centralises all response writing. This ensures a consistent envelope shape, correct headers, and a single place to handle encoding failures.

**Successful responses** are wrapped in a `SuccessResponse` envelope:

```go
// {"data": {"message": "Hello World!"}}
respond.WithOK(w, r, map[string]string{"message": "Hello World!"})
```

**Error responses** use sentinel `ResponseError` values with optional details:

```go
respond.WithError(w, r, respond.ErrNotFound)
respond.WithError(w, r, respond.ErrBadRequest.WithDetails("port must be between 1 and 65535"))
```

`WithDetails` returns a new `ResponseError` with the details field populated — it never mutates the sentinel value. This is safe because `WithDetails` uses a value receiver, which copies the struct before modifying it.

---

## The Server and Timeouts

```go
srv := &http.Server{
    Addr:         addr,
    Handler:      mux,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

Go's `http.Server` has no timeouts by default — a slow or malicious client can hold a connection open indefinitely. These three cover the main cases:

- **`ReadTimeout`** — maximum time to read the entire request including body
- **`WriteTimeout`** — maximum time to write the response
- **`IdleTimeout`** — how long a keep-alive connection can sit idle between requests

Without timeouts, a server is vulnerable to resource exhaustion from slow clients.

---

## Why `srvErr` is a Channel

`srv.ListenAndServe()` is a blocking call — it runs forever until the server stops. If called directly in `run`, nothing after it would execute while the server is running, which means we could never listen for shutdown signals simultaneously.

The solution is to run the server in a goroutine — a lightweight concurrent function managed by the Go runtime — and communicate its result back via a channel:

```go
srvErr := make(chan error, 1)

go func() {
    slog.InfoContext(ctx, "Starting server", "addr", srv.Addr)
    srvErr <- srv.ListenAndServe()
}()
```

A **channel** is a typed conduit for safely passing values between goroutines. `make(chan error, 1)` creates a buffered channel that can hold one `error` value without a receiver being ready. The `<-` operator sends a value into the channel (`srvErr <- err`) or receives from it (`err := <-srvErr`).

The buffer of `1` is important — without it the goroutine would block on `srvErr <- srv.ListenAndServe()` if `run` hasn't reached the `select` yet. With a buffer of 1, the send completes immediately and the goroutine exits cleanly.

---

## The `select` Statement

With the server running in a goroutine, `run` needs to wait for one of two things: the server failing unexpectedly, or a shutdown signal arriving. The `select` statement waits on multiple channel operations simultaneously and executes whichever fires first:

```go
select {
case err := <-srvErr:
    return err
case <-ctx.Done():
    slog.InfoContext(ctx, "Server shutting down")
}
```

`case err := <-srvErr` — receives from the error channel. This fires if `ListenAndServe` returns, which normally only happens on a startup error (e.g. port already in use).

`case <-ctx.Done()` — receives from the context's done channel. A `context.Context` carries a `Done()` channel that is closed when the context is cancelled. `signal.NotifyContext` cancels the context when `os.Interrupt` (`SIGINT`, i.e. Ctrl+C) is received.

`select` blocks until one case is ready, then executes that case and continues. It is Go's native mechanism for coordinating concurrent operations — no callbacks, no promises, no async/await.

---

## Graceful Shutdown

When a shutdown signal arrives we don't want to abruptly close connections mid-request. `srv.Shutdown` stops accepting new connections and waits for in-flight requests to complete before returning:

```go
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

return srv.Shutdown(shutdownCtx)
```

`context.WithTimeout` creates a derived context that automatically cancels after the given duration — giving in-flight requests 10 seconds to finish before the shutdown is forced.

Note that `context.Background()` is used as the parent rather than the signal context. The signal context is already cancelled at this point — using it as the parent would make `shutdownCtx` immediately cancelled too, giving requests no time to drain.

`defer shutdownCancel()` ensures the timeout's internal resources are released even if `Shutdown` returns early. `defer` runs a function call when the surrounding function returns, regardless of how it returns — a common Go pattern for cleanup that must always happen.

---

## Structured Logging with `log/slog`

Go 1.21 introduced `log/slog` as the standard structured logging package. Unlike the older `log` package which writes unstructured text, `slog` writes structured key-value pairs that can be output as JSON — making logs queryable, filterable, and compatible with observability platforms.

The core to understanding `slog` is the `slog.Handler` interface, which is the contract that every logging backend implements:

```go
type Handler interface {
    Enabled(context.Context, Level) bool
    Handle(context.Context, Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

Four methods, each with a specific job:

**`Enabled(ctx, level)`** — called _before_ a log record is constructed. If it returns false, `slog` skips the work of building the record entirely. This is how debug logging stays cheap when the level is set to info — the string formatting and attribute collection never happen.

**`Handle(ctx, record)`** — the core method. Receives the fully constructed log record and a context. A custom handler can inspect the context for request-scoped attributes and prepend them to the record before delegating to an inner handler. The context parameter is exactly why `slog.InfoContext(ctx, "msg")` exists — it threads the context all the way down to here.

**`WithAttrs(attrs)`** — returns a _new_ handler with those attributes baked in. This is what `slog.Logger.With("key", "val")` calls under the hood. The returned handler must include those attributes in every subsequent record. Critically, it must return a new handler, not mutate the existing one.

**`WithGroup(name)`** — returns a new handler that nests all subsequent attributes under a group name. Not used in this project but must be implemented to satisfy the interface.

### How it fits together

`slog.InfoContext(ctx, "msg", "key", "val")` eventually calls `handler.Handle(ctx, record)`. This project uses a `contextHandler` that sits between that call and the real output handler. It intercepts the record, pulls attributes from `ctx`, adds them, then passes the enriched record to the JSON handler:

```
┌─────────────────────────────────┐
│  slog.InfoContext(ctx, "msg")   │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│  contextHandler.Handle(ctx, r)  │
│  1. extract []slog.Attr from ctx│
│  2. prepend them to the record  │
│  3. delegate to inner handler   │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│  slog.JSONHandler (inner)       │
│  writes JSON to stdout          │
└─────────────────────────────────┘
```

### At the call site

The payoff is that call sites are completely unaware of the enrichment. Middleware stores request-scoped attributes in context, and any downstream code that logs with that context gets them for free:

```go
// middleware
ctx = logger.WithAttrs(ctx,
    slog.String("request_id", id),
    slog.String("method", r.Method),
)

// anywhere downstream — no knowledge of what attrs exist
slog.InfoContext(ctx, "processing request")
// output: {"msg":"processing request","request_id":"abc","method":"GET"}
```

### References

- The official `slog` package documentation covers the [Handler interface](https://pkg.go.dev/log/slog#Handler) contract in detail.
- The `slog` repo contains a thorough [Writing a handler](https://github.com/golang/example/blob/master/slog-handler-guide/README.md) guide.

## Middleware

The standard middleware signature in Go is `func(http.Handler) http.Handler`. It accepts a handler, wraps it, and returns a new handler. Because the input and output are the same type, middleware composes naturally — each layer calls `next.ServeHTTP(w, r)` to pass control inward, and code after that call runs on the way back out:

```
middlewareOne → middlewareTwo → handler → middlewareTwo → middlewareOne
```

### Chaining without dependencies

Nesting middleware manually (`one(two(three(handler)))`) is inside-out — the outermost call executes first, which is hard to read as the chain grows. Since Go 1.23, `slices.Backward` provides a clean way to build a chain type that applies middleware in declaration order:

```go
type middlewareChain []func(http.Handler) http.Handler

func (c middlewareChain) then(h http.Handler) http.Handler {
    for _, mw := range slices.Backward(c) {
        h = mw(h)
    }
    return h
}
```

`slices.Backward` iterates from last to first, wrapping inward, so the first middleware in the slice becomes the outermost handler. Routes then read top-to-bottom:

```go
baseChain := middlewareChain{middleware.RequestContext}
mux.Handle("GET /hello", baseChain.thenFunc(handle.Hello()))
```

For larger applications with many route groups and per-group middleware, Alex Edwards demonstrates a `Router` type that wraps `http.ServeMux` with nested group support — still no dependencies.

### Context enrichment via `r.WithContext`

`*http.Request` is immutable with respect to its context — `r.Context()` reads it, `r.WithContext(ctx)` returns a new request carrying the updated context. Middleware enriches context before calling `next.ServeHTTP`, making values available to every handler and middleware further down the chain without threading parameters through function signatures.

### Request ID vs Trace ID

A **request ID** is a correlation identifier scoped to a single request within a single service. It is generated or propagated at the edge of the service (typically by checking for an incoming `X-Request-ID` header from a load balancer, generating one via `crypto/rand` if absent) and set on the response header so callers can correlate.

A **trace ID** is an OpenTelemetry concept that identifies an operation across multiple services. It propagates via headers like `traceparent` and is managed by the OTel SDK, not application middleware. The two serve different purposes and coexist — a service will typically have both.

### References

- Alex Edwards: [Making and using HTTP Middleware](https://www.alexedwards.net/blog/making-and-using-middleware) — the standard pattern, chaining, and wrapping the mux.
- Alex Edwards: [Organize your Go middleware without dependencies](https://www.alexedwards.net/blog/organize-your-go-middleware-without-dependencies) — the `chain` type using `slices.Backward` and a `Router` type for nested groups.
- Mat Ryer's approach to the `func(http.Handler) http.Handler` signature and the `addRoutes` pattern informed the structure used here.
