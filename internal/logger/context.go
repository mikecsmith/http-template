package logger

import (
	"context"
	"log/slog"
)

type attrsKey struct{}

// Attrs returns the []slog.Attr stored in ctx, or nil if none exist.
func Attrs(ctx context.Context) []slog.Attr {
	if l, ok := ctx.Value(attrsKey{}).([]slog.Attr); ok {
		return l
	}
	return nil
}

// WithAttrs returns a new context carrying the provided attrs
// appended to any attrs already in ctx.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	a := append(Attrs(ctx), attrs...)
	return context.WithValue(ctx, attrsKey{}, a)
}
