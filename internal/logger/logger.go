// Package logger provides context-aware structured logging built on log/slog.
//
// contextHandler wraps an inner slog.Handler (typically slog.JSONHandler) and
// implements the slog.Handler interface. On each call to Handle, it reads
// []slog.Attr from the context and prepends them to the log record before
// delegating to the inner handler. Enabled, WithAttrs, and WithGroup delegate
// directly — contextHandler has no opinion on log levels or grouping, it only
// injects context-carried attributes.
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
	"log/slog"
	"os"
)

// Init configures the global slog logger with a JSON handler writing to stdout and returns a default logger
func Init() {
	inner := slog.NewJSONHandler(os.Stdout, nil)
	h := NewContextHandler(inner)
	slog.SetDefault(slog.New(h))
}
