package handle

import (
	"net/http"

	"github.com/mikecsmith/httplab/internal/respond"
)

func NotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.WithError(w, r, respond.ErrNotFound)
	}
}
