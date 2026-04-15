package logger

import (
	"context"
	"log/slog"
	"slices"
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
//
// slices.Clip forces append to allocate a new backing array rather
// than writing into any spare capacity shared with sibling contexts.
// Without it, two goroutines deriving from the same parent context
// would both write into the same underlying slice — corrupting each
// other's log fields silently. The extra allocation is cheap, and
// attrs are written rarely (once per middleware stage) compared to
// how often they are read.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	a := append(slices.Clip(Attrs(ctx)), attrs...)
	return context.WithValue(ctx, attrsKey{}, a)
}
