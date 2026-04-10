package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mikecsmith/http-template/internal/middleware"
)

func TestSecureHeaders(t *testing.T) {
	// Baseline: running the middleware against a no-op inner handler
	// should leave every expected header set to its documented value.
	t.Run("sets the expected baseline headers", func(t *testing.T) {
		want := map[string]string{
			middleware.HeaderContentTypeOptions:        middleware.ValueContentTypeOptions,
			middleware.HeaderContentSecurityPolicy:     middleware.ValueContentSecurityPolicy,
			middleware.HeaderReferrerPolicy:            middleware.ValueReferrerPolicy,
			middleware.HeaderStrictTransportSecurity:   middleware.ValueStrictTransportSecurity,
			middleware.HeaderCrossOriginResourcePolicy: middleware.ValueCrossOriginResourcePolicy,
		}

		handler := middleware.SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, req)

		for name, wantVal := range want {
			if got := w.Header().Get(name); got != wantVal {
				t.Errorf("%s = %q, want %q", name, got, wantVal)
			}
		}
	})

	// A handler that writes a header *after* SecureHeaders set one
	// must win — this is the intended override path for routes that
	// need a relaxed CSP, a different CORP, etc.
	t.Run("inner handler can override a header", func(t *testing.T) {
		const custom = "default-src 'self'"

		handler := middleware.SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(middleware.HeaderContentSecurityPolicy, custom)
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, req)

		if got := w.Header().Get(middleware.HeaderContentSecurityPolicy); got != custom {
			t.Errorf("CSP = %q, want inner override %q", got, custom)
		}
	})

	// Headers should be present even on responses the inner handler
	// doesn't explicitly write — e.g. a bare WriteHeader(204).
	t.Run("headers are set even with empty response body", func(t *testing.T) {
		handler := middleware.SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, req)

		if got := w.Header().Get(middleware.HeaderContentTypeOptions); got != middleware.ValueContentTypeOptions {
			t.Errorf("X-Content-Type-Options = %q, want %q", got, middleware.ValueContentTypeOptions)
		}
	})
}
