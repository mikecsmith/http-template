package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/mikecsmith/httplab/internal/logger"
)

// testHandler returns a slog.Logger backed by a contextHandler writing
// text to a buffer. The buffer is returned for assertion.
func testHandler(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)
	h := logger.NewContextHandler(inner)
	return slog.New(h), &buf
}

// --- Context storage ---

// TestAttrs verifies that []slog.Attr values can be stored in and
// retrieved from a context.Context, and that storing does not mutate
// parent contexts.
func TestAttrs(t *testing.T) {
	// A bare context has no attrs stored — Attrs should return nil,
	// which is the zero value for a slice and safe to range over.
	t.Run("returns nil when context has no attrs", func(t *testing.T) {
		got := logger.Attrs(context.Background())
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	// WithAttrs stores attrs in context. Attrs retrieves them.
	// This round-trips a single attr and checks both key and value
	// survive storage.
	t.Run("returns stored attrs", func(t *testing.T) {
		k := "request_id"
		v := "abc-123"
		ctx := logger.WithAttrs(context.Background(),
			slog.String(k, v),
		)
		got := logger.Attrs(ctx)

		if len(got) != 1 {
			t.Fatalf("got %d attrs, want 1", len(got))
		}
		if got[0].Key != k {
			t.Errorf("expected key %s, got %s", k, got[0].Key)
		}
		if got[0].Value.String() != v {
			t.Errorf("expected value %s, got %s", v, got[0].Value.String())
		}
	})

	// context.WithValue returns a new context — the parent is never
	// modified. This test proves that adding attrs to a child context
	// does not affect the parent, which is critical for concurrent
	// request handling where contexts fork per-request.
	t.Run("appends to existing attrs without mutating parent context", func(t *testing.T) {
		parent := logger.WithAttrs(context.Background(),
			slog.String("first", "one"),
		)
		child := logger.WithAttrs(parent,
			slog.String("second", "two"),
		)

		parentAttrs := logger.Attrs(parent)
		childAttrs := logger.Attrs(child)

		if len(parentAttrs) != 1 {
			t.Fatalf("got %d attrs, want 1", len(parentAttrs))
		}
		if len(childAttrs) != 2 {
			t.Fatalf("got %d attrs, want 2", len(childAttrs))
		}
	})
}

// --- Handler behaviour ---

// TestContextHandler verifies that contextHandler correctly implements
// slog.Handler: extracting attrs from context in Handle, delegating
// level decisions in Enabled, and preserving the wrapping through
// WithAttrs and WithGroup.
func TestContextHandler(t *testing.T) {
	// Baseline: the handler works as a normal slog.Handler even when
	// no attrs are stored in context. This catches nil-safety issues
	// in the Handle method.
	t.Run("logs without context attrs produce normal output", func(t *testing.T) {
		l, buf := testHandler(t)

		l.InfoContext(context.Background(), "bare message")

		if !bytes.Contains(buf.Bytes(), []byte("bare message")) {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	// The core behaviour: attrs stored in context via logger.WithAttrs
	// appear in log output without the call site doing anything beyond
	// passing ctx. This is the whole point of the custom handler.
	t.Run("context attrs appear in log output", func(t *testing.T) {
		l, buf := testHandler(t)

		ctx := logger.WithAttrs(context.Background(),
			slog.String("request_id", "req-456"),
			slog.String("method", "GET"),
		)
		l.InfoContext(ctx, "request started")

		for _, want := range []string{"req-456", "GET"} {
			if !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Errorf("expected %q in output, got: %s", want, buf.String())
			}
		}
	})

	// l.With("key", "val") calls handler.WithAttrs under the hood.
	// If WithAttrs returns the bare inner handler instead of a new
	// contextHandler, baked-in attrs still work but context extraction
	// silently stops. This test catches that — both enrichment paths
	// must produce output.
	t.Run("handler-level attrs and context attrs coexist", func(t *testing.T) {
		l, buf := testHandler(t)
		hl := l.With("service", "httplab")

		ctx := logger.WithAttrs(context.Background(),
			slog.String("request_id", "req-789"),
		)
		hl.InfoContext(ctx, "request started")

		if !bytes.Contains(buf.Bytes(), []byte("service=httplab")) {
			t.Errorf("expected service=httplab in output. Output: %s", buf.String())
		}
		if !bytes.Contains(buf.Bytes(), []byte("request_id=req-789")) {
			t.Errorf("expected request_id=req-789 in output. Output: %s", buf.String())
		}
	})

	// Same principle as coexist: WithGroup must return a new
	// contextHandler wrapping the grouped inner handler, not the
	// bare grouped handler. The text handler renders grouped attrs
	// with a dot prefix (request.method), which we assert on.
	t.Run("WithGroup delegates to inner handler", func(t *testing.T) {
		var buf bytes.Buffer
		inner := slog.NewTextHandler(&buf, nil)
		h := logger.NewContextHandler(inner)
		nh := h.WithGroup("request")

		ctx := logger.WithAttrs(context.Background(), slog.String("method", "GET"))
		l := slog.New(nh)
		l.InfoContext(ctx, "request started")

		if !bytes.Contains(buf.Bytes(), []byte("request.method=GET")) {
			t.Errorf("expected request.method=GET in output. Output: %s", buf.String())
		}
	})

	// contextHandler has no opinion on log levels — it delegates
	// Enabled entirely to the inner handler. This test configures the
	// inner handler to accept only Warn and above, then verifies that
	// contextHandler respects that threshold.
	t.Run("Enabled delegates to inner handler", func(t *testing.T) {
		inner := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
		h := logger.NewContextHandler(inner)

		if h.Enabled(context.Background(), slog.LevelInfo) {
			t.Fatal("expected Enabled to return false for LevelInfo")
		}
		if !h.Enabled(context.Background(), slog.LevelWarn) {
			t.Fatal("expected Enabled to return true for LevelWarn")
		}
	})
}
