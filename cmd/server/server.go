package main

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/mikecsmith/http-template/internal/config"
)

// NewServer is the single place where the HTTP handler graph is
// assembled. It creates the mux, hands it to addRoutes for route
// registration, wraps the result in [http.TimeoutHandler] so
// cfg.RequestTimeout strictly caps handler execution, and finally
// wraps everything in otelhttp so every request — including
// unknown paths, health probes, and timeouts — gets counted in the
// standard net/http server-side metric set
// (http.server.request.duration, http.server.active_requests, etc.).
//
// Handler graph, outermost → innermost:
//
//  1. otelhttp — per-request metrics. Sits outermost so timeouts
//     produced by the next layer are still observed.
//  2. http.TimeoutHandler — enforces cfg.RequestTimeout on handler
//     execution. A handler that exceeds the deadline has its
//     context cancelled and a 503 "request timeout" response
//     written in its place. This is the right home for the cap
//     because http.Server.ReadTimeout only governs body read, not
//     handler runtime — setting ReadTimeout to RequestTimeout (as
//     this template originally did) does NOT bound handler
//     execution no matter what the flag description says.
//  3. mux + the per-route middleware chain — request context,
//     logging, secure headers. Per-route middleware stays in
//     routes.go because its ordering choices are tied to each
//     route's chain, not to server-level hardening.
//
// When MetricsExporter is "none" the otelhttp wrapper is essentially
// free: it records into the OTel global MeterProvider, which is a
// no-op provider until metrics.Init installs something else.
func NewServer(cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, cfg)
	timeoutHandler := http.TimeoutHandler(mux, cfg.RequestTimeout, "request timeout")
	return otelhttp.NewHandler(timeoutHandler, cfg.OtelServiceName)
}
