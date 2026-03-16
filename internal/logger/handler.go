package logger

import (
	"context"
	"log/slog"
)

// contextHandler wraps an inner slog.Handler, extracting attrs from
// the context and prepending them to every record before delegation.
type contextHandler struct {
	inner slog.Handler
}

// NewContextHandler returns a handler that extracts attrs from context
// before delegating to inner.
func NewContextHandler(inner slog.Handler) *contextHandler {
	return &contextHandler{inner: inner}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs := Attrs(ctx); attrs != nil {
		r.AddAttrs(attrs...)
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewContextHandler(h.inner.WithAttrs(attrs))
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return NewContextHandler(h.inner.WithGroup(name))
}
