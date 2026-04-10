package main

import (
	"net/http"
	"slices"

	"github.com/mikecsmith/http-template/internal/config"
	"github.com/mikecsmith/http-template/internal/handle"
	"github.com/mikecsmith/http-template/internal/middleware"
)

// middlewareChain is an ordered list of middleware. The [then] method
// applies them so that the first element in the slice is the outermost
// handler — matching declaration order rather than the inside-out
// nesting you get from manual wrapping. Based on Alex Edwards' pattern
// using [slices.Backward].
type middlewareChain []func(http.Handler) http.Handler

// then wraps h with every middleware in the chain, outermost first.
func (c middlewareChain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

// addRoutes is the single place where the full API surface is defined.
// A single middleware chain applies to every route, including health
// probes. Ordering matters:
//
//  1. RequestContext — outermost. Generates/propagates the request ID
//     and seeds the logger context attrs. Must run first so downstream
//     middleware (Logging) can read the attrs it injects.
//  2. Logging — second. Sees every request, including ones a later
//     middleware short-circuits (future auth, rate limit, etc.). With
//     LogLevel defaulting to Error, successful probe traffic is
//     constructed cheaply and dropped at the slog handler via the
//     Enabled guard inside the middleware, so the cost on /healthz
//     and /readyz is a few hundred nanoseconds per hit.
//  3. SecureHeaders — innermost. Sets the response header baseline
//     before the handler runs.
//
// Per-request metrics are produced by otelhttp, which wraps this whole
// chain in NewServer — outside the chain, so metrics see every request
// regardless of what the chain does.
func addRoutes(mux *http.ServeMux, _ config.Config) {
	chain := middlewareChain{
		middleware.RequestContext,
		middleware.Logging,
		middleware.SecureHeaders,
	}

	mux.Handle("GET /healthz", chain.then(handle.Healthz()))
	mux.Handle("GET /readyz", chain.then(handle.Readyz()))
	mux.Handle("GET /hello", chain.then(handle.HelloWorldGet()))
	mux.Handle("POST /hello", chain.then(handle.HelloWorldPost()))
	mux.Handle("/", chain.then(handle.NotFound()))
}
