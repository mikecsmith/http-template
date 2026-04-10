package main

import (
	"net/http"
	"slices"

	"github.com/mikecsmith/httplab/internal/handle"
	"github.com/mikecsmith/httplab/internal/middleware"
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
// Health probes are registered without middleware so Kubernetes probes
// stay lightweight. API routes use the apiChain for request context
// enrichment (request ID, structured logging attributes).
func addRoutes(mux *http.ServeMux, _ config) {
	globalChain := middlewareChain{}
	apiChain := append(globalChain, middleware.RequestContext)
	mux.Handle("GET /healthz", handle.Healthz())
	mux.Handle("GET /readyz", handle.Readyz())
	mux.Handle("GET /hello", apiChain.then(handle.HelloWorldGet()))
	mux.Handle("POST /hello", apiChain.then(handle.HelloWorldPost()))
	mux.Handle("/", apiChain.then(handle.NotFound()))
}
