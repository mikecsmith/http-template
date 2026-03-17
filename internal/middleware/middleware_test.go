package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mikecsmith/httplab/internal/logger"
	"github.com/mikecsmith/httplab/internal/middleware"
)

// captureAttrs returns a handler that records whatever logger attrs
// are present in the request context at the time it executes.
func captureAttrs(t *testing.T, got *[]slog.Attr) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		*got = logger.Attrs(r.Context())
	}
}

func TestLoggerContext(t *testing.T) {
	// The middleware must store method and path as slog.Attr values
	// in the request context before the handler executes. The handler
	// receives the enriched context — it never needs to extract these
	// values from the request itself.
	t.Run("injects method and path attrs into context", func(t *testing.T) {
		var got []slog.Attr
		handler := middleware.LoggerContext(captureAttrs(t, &got))

		req := httptest.NewRequest(http.MethodPost, "/hello", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		// KOAN: assert that got contains exactly 2 attrs,
		if len(got) != 2 {
			t.Fatalf("Expected 2 attrs, got: %d", len(got))
		}

		if got[0].Key != "method" || got[0].Value.String() != "POST" {
			t.Errorf("Expected method=POST, got: %v", got[0])
		}
		if got[1].Key != "path" || got[1].Value.String() != "/hello" {
			t.Errorf("Expected path=/hello, got: %v", got[1])
		}
	})

	// A request to a different method and path must produce different
	// attr values. This catches hardcoded values.
	t.Run("attrs reflect the actual request", func(t *testing.T) {
		var got []slog.Attr
		handler := middleware.LoggerContext(captureAttrs(t, &got))

		req := httptest.NewRequest(http.MethodDelete, "/users/42", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		expected := map[string]string{
			"method": "DELETE",
			"path":   "/users/42",
		}

		for _, attr := range got {
			want, ok := expected[attr.Key]
			if !ok {
				t.Errorf("got unexpected attr key: %s", attr.Key)
				continue
			}
			if attr.Value.String() != want {
				t.Errorf("attr %s: expected %s, got %s", attr.Key, want, attr.Value.String())
			}
		}
	})
}
