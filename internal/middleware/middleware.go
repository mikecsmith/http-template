// Package middleware provides HTTP middleware for the server.
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/mikecsmith/httplab/internal/logger"
)

func LoggerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := logger.WithAttrs(
			r.Context(),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
