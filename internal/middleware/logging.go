package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Attr keys for the fields this middleware adds to each request log
// record. Exported so tests and downstream log consumers can reference
// them by name rather than duplicating string literals.
const (
	AttrStatus   = "status"
	AttrBytes    = "bytes"
	AttrDuration = "duration"
)

// Logging logs each HTTP request after the handler returns. The log
// record carries the response status, the number of bytes written, and
// the wall-clock duration the handler ran for. The log level adapts to
// the response status: Info for 2xx/3xx, Warn for 4xx, Error for 5xx,
// so a default Info-level logger surfaces client and server errors
// prominently without drowning the stream in noise.
//
// Logging relies on the request context already carrying request_id,
// method, and path attrs — which RequestContext injects — so this
// middleware must appear *after* RequestContext in a chain. In terms
// of middlewareChain ordering that means listed later in the slice,
// which makes Logging the inner wrapper and RequestContext the outer.
// In other words: this middleware should be the closest one to the
// handler itself.
//
// The middleware wraps the response writer so it can observe the
// status code and bytes written. The wrapper implements Unwrap so
// http.ResponseController can reach the underlying writer's Flush,
// Hijack, etc. interfaces — handlers using those features work
// transparently.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		// Guard the slog call with Enabled so a filtered-out record
		// costs nothing more than a level comparison. slog.Log itself
		// would still construct the Record and evaluate every attr
		// before dropping it at the handler. The saving is marginal for
		// three attrs; it earns its keep as soon as the middleware
		// grows richer fields (remote_addr, user_agent, trace IDs).
		status := rec.Status()
		level := levelForRequest(r.URL.Path, status)
		ctx := r.Context()
		if !slog.Default().Enabled(ctx, level) {
			return
		}
		slog.Log(ctx, level, "request",
			slog.Int(AttrStatus, status),
			slog.Int(AttrBytes, rec.bytes),
			slog.Duration(AttrDuration, time.Since(start)),
		)
	})
}

// levelForRequest maps a request path + response status to a slog
// level. 5xx is something the operator almost certainly wants to see;
// 4xx is worth noting but usually client error; everything else is
// routine. Successful hits to kubelet probe endpoints are demoted to
// Debug — on a live cluster these fire every couple of seconds and
// would otherwise drown out real traffic. A failing probe still
// surfaces at Warn/Error via the status-based branches.
func levelForRequest(path string, status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	case isProbePath(path):
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// isProbePath reports whether the request targets a kubelet probe
// endpoint. Kept as a tiny helper so the set of probe paths lives in
// one place.
func isProbePath(path string) bool {
	return path == "/healthz" || path == "/readyz"
}

// responseRecorder wraps http.ResponseWriter to capture the status code
// and the byte count written by a handler. The zero value is usable
// once the embedded ResponseWriter is set.
//
// Only the first WriteHeader call is recorded, matching net/http's
// behaviour of ignoring subsequent calls. If the handler writes a body
// without calling WriteHeader, we set status to 200 on the first Write
// to mirror what the stdlib emits on the wire. If the handler neither
// writes a body nor calls WriteHeader, Status() defaults to 200 to
// match the implicit response the stdlib sends when the handler
// returns.
//
// Unwrap returns the underlying ResponseWriter so that
// http.ResponseController can reach through to optional interfaces
// like http.Flusher or http.Hijacker. This is the Go 1.20+ idiom for
// wrapping response writers without breaking feature detection.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
