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

func TestAttrs(t *testing.T) {
	t.Run("returns nil when context has no attrs", func(t *testing.T) {
		got := logger.Attrs(context.Background())
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

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

	t.Run("appends to existing attrs without mutating parent context", func(t *testing.T) {
		parent := logger.WithAttrs(context.Background(),
			slog.String("first", "one"),
		)
		child := logger.WithAttrs(parent,
			slog.String("second", "two"),
		)

		parentAttrs := logger.Attrs(parent)
		childAttrs := logger.Attrs(child)
		_ = parentAttrs
		_ = childAttrs

		t.Skip("KOAN: assert parentAttrs has length 1 and childAttrs has length 2")
	})
}

// --- Handler behaviour ---

func TestContextHandler(t *testing.T) {
	t.Run("logs without context attrs produce normal output", func(t *testing.T) {
		l, buf := testHandler(t)

		l.InfoContext(context.Background(), "bare message")

		if !bytes.Contains(buf.Bytes(), []byte("bare message")) {
			t.Errorf("expected message in output, got: %s", buf.String())
		}
	})

	t.Run("context attrs appear in log output", func(t *testing.T) {
		l, buf := testHandler(t)

		ctx := logger.WithAttrs(context.Background(),
			slog.String("request_id", "req-456"),
			slog.String("method", "GET"),
		)
		l.InfoContext(ctx, "request started")

		_ = buf

		t.Skip("KOAN: assert that buf.String() contains both \"req-456\" and \"GET\"")
	})

	t.Run("handler-level attrs and context attrs coexist", func(t *testing.T) {
		l, buf := testHandler(t)

		_ = l
		_ = buf

		t.Skip("KOAN: create a derived logger using l.With() that bakes in a \"service\" attr with value \"httplab\", then log with context attrs and assert both appear in output")
	})

	t.Run("WithGroup delegates to inner handler", func(t *testing.T) {
		_, buf := testHandler(t)

		_ = buf

		t.Skip("KOAN: create a handler with NewContextHandler, call WithGroup(\"request\"), build a logger, log with a \"method\" attr, assert output contains \"request.method\"")
	})

	t.Run("Enabled delegates to inner handler", func(t *testing.T) {
		inner := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
		h := logger.NewContextHandler(inner)

		_ = h

		t.Skip("KOAN: assert h.Enabled returns false for LevelInfo and true for LevelWarn")
	})
}
