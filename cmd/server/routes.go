package main

import (
	"net/http"

	"github.com/mikecsmith/httplab/internal/handle"
)

func addRoutes(mux *http.ServeMux) {
	mux.Handle("GET /hello", handle.Hello())
}
