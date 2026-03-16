// Package handle contains handlers for various API operations
package handle

import (
	"net/http"

	"github.com/mikecsmith/httplab/internal/respond"
)

// Hello returns a simple hello world response.
func Hello() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{
			"message": "Hello World!",
		}
		respond.WithOK(w, r, data)
	}
}
