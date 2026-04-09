package main

import (
	"net/http"
	"slices"

	"github.com/mikecsmith/httplab/internal/handle"
	"github.com/mikecsmith/httplab/internal/middleware"
)

type middlewareChain []func(http.Handler) http.Handler

func (c middlewareChain) then(h http.Handler) http.Handler {
	for _, mw := range slices.Backward(c) {
		h = mw(h)
	}
	return h
}

func addRoutes(mux *http.ServeMux) {
	baseChain := middlewareChain{middleware.RequestContext}
	mux.Handle("GET /healthz", baseChain.then(handle.Healthz()))
	mux.Handle("GET /readyz", baseChain.then(handle.Healthz()))
}
