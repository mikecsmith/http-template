package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mikecsmith/httplab/internal/logger"
	"github.com/mikecsmith/httplab/internal/middleware"
)

const (
	headerRequestID = middleware.RequestIDHeader
	attrMethod      = middleware.AttrMethod
	attrPath        = middleware.AttrPath
	attrRequestID   = middleware.AttrRequestID
)

// captureAttrs returns a handler that records whatever logger attrs
// are present in the request context at the time it executes.
func captureAttrs(t *testing.T, got *[]slog.Attr) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		*got = logger.Attrs(r.Context())
	}
}

// attrMap converts a slice of slog.Attr into a map for easier assertion.
func attrMap(attrs []slog.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Key] = a.Value.String()
	}
	return m
}

func TestRequestContext(t *testing.T) {
	// The combined middleware must inject method, path, and request_id
	// as slog attrs into the request context. This is the core contract
	// — downstream handlers get enriched logging for free via ctx.
	t.Run("injects method, path, and request_id attrs", func(t *testing.T) {
		var got []slog.Attr
		handler := middleware.RequestContext(captureAttrs(t, &got))

		req := httptest.NewRequest(http.MethodPost, "/hello", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		if len(got) != 3 {
			t.Fatalf("expected 3 attrs, got %d", len(got))
		}

		m := attrMap(got)
		if m[attrMethod] != "POST" {
			t.Errorf("expected method=POST, got %s", m[attrMethod])
		}
		if m[attrPath] != "/hello" {
			t.Errorf("expected path=/hello, got %s", m[attrPath])
		}
		if len(m[attrRequestID]) != 32 {
			t.Errorf("expected 32-char request_id, got %d chars: %s", len(m[attrRequestID]), m[attrRequestID])
		}
	})

	// When an upstream service (load balancer, API gateway) sets
	// X-Request-ID, the middleware must propagate it rather than
	// generating a new one. This preserves correlation across services.
	t.Run("respects existing X-Request-ID header", func(t *testing.T) {
		var got []slog.Attr
		handler := middleware.RequestContext(captureAttrs(t, &got))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerRequestID, "upstream-id-123")
		handler.ServeHTTP(httptest.NewRecorder(), req)

		m := attrMap(got)
		if m[attrRequestID] != "upstream-id-123" {
			t.Errorf("expected request_id=upstream-id-123, got %s", m[attrRequestID])
		}
	})

	// The request ID must appear on the response header so callers
	// can correlate their request with server-side logs.
	t.Run("sets request ID on response header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		handler := middleware.RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		handler.ServeHTTP(w, req)

		got := w.Header().Get(headerRequestID)
		if got == "" {
			t.Error("expected X-Request-ID on response header, got empty string")
		}
		if len(got) != 32 {
			t.Errorf("expected 32-char request ID on response, got %d chars: %s", len(got), got)
		}
	})

	// When an upstream ID is provided, the response header must echo
	// it back rather than a newly generated one.
	t.Run("response header echoes upstream request ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerRequestID, "upstream-id-456")

		handler := middleware.RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		handler.ServeHTTP(w, req)

		got := w.Header().Get(headerRequestID)
		if got != "upstream-id-456" {
			t.Errorf("expected upstream-id-456 on response header, got %s", got)
		}
	})

	// Two requests without an upstream ID must produce different
	// generated IDs. Not a proof of randomness, but catches the
	// obvious bug of returning a constant or reusing a seed.
	t.Run("generates unique IDs across requests", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		var ids []string

		for range 2 {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			middleware.RequestContext(inner).ServeHTTP(w, req)
			ids = append(ids, w.Header().Get(headerRequestID))
		}

		if ids[0] == ids[1] {
			t.Errorf("expected unique IDs, got %s twice", ids[0])
		}
	})
}
