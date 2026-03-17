// Package middleware provides HTTP middleware for the server.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/mikecsmith/httplab/internal/logger"
)

const (
	RequestIDHeader = "X-Request-ID"
	AttrMethod      = "method"
	AttrPath        = "path"
	AttrRequestID   = "request_id"
)

// RequestContext combines request ID generation and logger context
// enrichment into a single middleware. It generates or propagates a
// request ID, sets it on the response header, and stores method, path,
// and request ID as slog attrs in the request context.
func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := idFromRequest(r)
		r.Header.Set(RequestIDHeader, id)
		w.Header().Set(RequestIDHeader, id)

		ctx := logger.WithAttrs(
			r.Context(),
			slog.String(AttrMethod, r.Method),
			slog.String(AttrPath, r.URL.Path),
			slog.String(AttrRequestID, r.Header.Get(RequestIDHeader)))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	s := make([]byte, 16)
	if _, err := rand.Read(s); err != nil {
		panic("System unable to provide random numbers")
	}
	return hex.EncodeToString(s)
}

func idFromRequest(r *http.Request) string {
	if h := r.Header.Get(RequestIDHeader); h != "" {
		return h
	}
	return generateRequestID()
}
