// Package handle contains handlers for various API operations
package handle

import (
	"net/http"

	"github.com/mikecsmith/httplab/internal/respond"
)

// Healthz returns a 200 OK response to show the API is healthy.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.WithOK(w, r, nil)
	}
}
