package main

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/mikecsmith/httplab/internal/config"
)

// NewServer is the single place where the HTTP handler graph is
// assembled. It creates the mux, hands it to addRoutes for route
// registration, and wraps the whole thing in otelhttp so every
// request — including unknown paths and health probes — gets counted
// in the standard net/http server-side metric set
// (http.server.request.duration, http.server.active_requests, etc.).
//
// otelhttp sits outside the per-route middleware chain because the
// metrics it produces should reflect every request, not just the ones
// that make it past in-chain rejections (auth, rate limit, etc.). Logs,
// by contrast, sit inside the chain and default to Error level — see
// routes.go for the full ordering.
//
// When MetricsExporter is "none" this wrapper is essentially free:
// otelhttp records into the OTel global MeterProvider, which is a no-op
// provider until metrics.Init installs something else.
func NewServer(cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, cfg)
	return otelhttp.NewHandler(mux, "httplab")
}
