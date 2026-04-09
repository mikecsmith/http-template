package handle

import (
	"net/http"

	"github.com/mikecsmith/httplab/internal/respond"
)

// Readyz should be extended to inject anything required to check the service is ready to serve traffic
func Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.WithOK(w, r, nil)
	}
}
