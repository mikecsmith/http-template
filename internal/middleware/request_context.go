package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/mikecsmith/http-template/internal/logger"
)

// RequestContext middleware constants. The header name follows the
// widely-used X-Request-ID convention; the attr names are the keys
// under which method, path, and request ID appear in structured logs.
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

// idFromRequest returns the request ID carried on the incoming request,
// or generates a new one if none is present. Propagating an upstream ID
// preserves correlation across services (load balancers, API gateways).
func idFromRequest(r *http.Request) string {
	if h := r.Header.Get(RequestIDHeader); h != "" {
		return h
	}
	return generateRequestID()
}

// generateRequestID returns a 32-character hex-encoded random ID. The
// caller's assumption is that crypto/rand always succeeds; if it
// doesn't, something is very wrong with the host and panicking is the
// correct response.
func generateRequestID() string {
	s := make([]byte, 16)
	if _, err := rand.Read(s); err != nil {
		panic("System unable to provide random numbers")
	}
	return hex.EncodeToString(s)
}
