// Package logger provides context-aware structured logging built on log/slog.
//
// [Init] is the entry point. It builds a [slog.JSONHandler] writing to
// the supplied io.Writer at the supplied level, wraps it in a
// contextHandler, and installs the result as slog's default logger via
// [slog.SetDefault]. The level parameter is a [slog.Leveler] rather
// than a plain [slog.Level] so callers can pass a [*slog.LevelVar] and
// change the level at runtime (e.g. via a signal handler or admin
// endpoint) without rebuilding the handler. Passing a plain Level
// works too — Level satisfies Leveler by returning itself.
//
// contextHandler wraps an inner slog.Handler (typically slog.JSONHandler) and
// implements the slog.Handler interface. On each call to Handle, it reads
// []slog.Attr from the context and prepends them to the log record before
// delegating to the inner handler. Enabled, WithAttrs, and WithGroup delegate
// directly — contextHandler has no opinion on log levels or grouping, those
// concerns live on the inner handler and flow through unchanged.
//
// Attributes are stored in context as []slog.Attr via WithAttrs and retrieved
// via Attrs. This keeps call sites simple: pass ctx to any slog.*Context
// function and the request-scoped fields appear automatically.
//
// The pattern works anywhere a context flows, not just HTTP handlers:
//
//	// HTTP middleware enriches context for the duration of a request:
//	ctx = logger.WithAttrs(ctx,
//	    slog.String("request_id", id),
//	    slog.String("method", r.Method),
//	)
//	slog.InfoContext(ctx, "request started")
//	// output: {"msg":"request started","request_id":"abc","method":"GET"}
//
//	// A CLI command can do the same for the duration of a run:
//	ctx = logger.WithAttrs(ctx, slog.String("command", "sync"))
//	slog.InfoContext(ctx, "starting sync")
//	// output: {"msg":"starting sync","command":"sync"}
//
//	// A background worker can carry job-scoped attributes:
//	ctx = logger.WithAttrs(ctx, slog.String("job_id", job.ID))
//	slog.InfoContext(ctx, "processing")
//	// output: {"msg":"processing","job_id":"job_789"}
//
// In every case the calling code just passes ctx. It does not need to know
// which attributes have been attached upstream or how the handler extracts them.
package logger

import (
	"io"
	"log/slog"
)

// Init configures the global slog logger with a JSON handler writing
// to w at the given level. level is a [slog.Leveler] rather than a
// plain [slog.Level] so callers can pass a [*slog.LevelVar] and change
// the level at runtime (e.g. via a signal or admin endpoint) without
// rebuilding the handler. Passing a plain [slog.Level] also works —
// the Level type satisfies Leveler by returning itself.
func Init(w io.Writer, level slog.Leveler) {
	inner := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	h := NewContextHandler(inner)
	slog.SetDefault(slog.New(h))
}
