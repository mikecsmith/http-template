package main

import (
	"net/http"
	"slices"

	"github.com/mikecsmith/httplab/internal/config"
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
// Two middleware chains are used:
//
//   - globalChain applies to every route, including health probes. It
//     carries baseline concerns that every response should have —
//     currently just SecureHeaders.
//   - apiChain extends globalChain with request-context enrichment
//     (request ID, structured logging attrs). Health probes skip this
//     extra layer so K8s probe traffic stays lightweight.
func addRoutes(mux *http.ServeMux, _ config.Config) {
	globalChain := middlewareChain{middleware.SecureHeaders}
	apiChain := append(globalChain, middleware.RequestContext)

	mux.Handle("GET /healthz", globalChain.then(handle.Healthz()))
	mux.Handle("GET /readyz", globalChain.then(handle.Readyz()))
	mux.Handle("GET /hello", apiChain.then(handle.HelloWorldGet()))
	mux.Handle("POST /hello", apiChain.then(handle.HelloWorldPost()))
	mux.Handle("/", apiChain.then(handle.NotFound()))
}
