// Package middleware contains middleware which wrap http.HandlerFunc's to injection additional behaviour
package middleware

import "net/http"

func LoggerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Anything here is called before the handler executes
		next.ServeHTTP(w, r)
		// Anything here is called after the handler executes
	})
}
