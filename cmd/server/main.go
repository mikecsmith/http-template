// Package main is the entry point for the HTTP server.
//
// main is kept deliberately minimal — it calls [run] with the real OS
// dependencies and translates any error into a non-zero exit code. All
// startup logic lives in [run] so the entire program can be exercised
// from tests by calling run directly with controlled inputs.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/mikecsmith/http-template/internal/config"
	"github.com/mikecsmith/http-template/internal/logger"
	"github.com/mikecsmith/http-template/internal/metrics"
)

// Build metadata. version is always set via -ldflags at release time
// (see .goreleaser.yaml). commit and date are also set via -ldflags on
// release builds; on local `go build` they fall back to the VCS info
// Go embeds automatically in BuildInfo.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// On release builds ldflags overwrites these vars before init runs,
	// so we only fill in from BuildInfo when they still have their dev
	// defaults — meaning this is a plain `go build` from a git checkout.
	if commit != "none" && date != "unknown" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if commit == "none" && len(s.Value) >= 8 {
				commit = s.Value[:8]
			}
		case "vcs.time":
			if date == "unknown" {
				if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
					date = t.Format(time.DateOnly)
				}
			}
		}
	}
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdout, os.Interrupt, nil); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// run is the real entry point. It takes OS fundamentals as arguments so
// that tests can call it with controlled values — fake args, env vars,
// and output writers — without mutating global state.
//
// The sig parameter controls which OS signal triggers graceful shutdown.
// The ready callback, if non-nil, is invoked once the listener is bound
// with the resolved address — this lets tests use port "0" and then
// discover the actual port the kernel picked. Production callers pass
// nil.
func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	out io.Writer,
	sig os.Signal,
	ready func(addr string),
) error {
	cfg, err := config.ParseConfig(args, getenv)
	if err != nil {
		return err
	}

	logger.Init(out, cfg.LogLevel)

	// metrics.Init installs a MeterProvider into the OTel global when
	// enabled, so that otelhttp (wired in NewServer) can pick it up
	// without having to be handed a provider directly. The returned
	// shutdown func is non-nil on the disabled path too; it is safe
	// to call either way.
	metricsShutdown, err := metrics.Init(cfg.MetricsEnabled)
	if err != nil {
		return fmt.Errorf("metrics init: %w", err)
	}

	ctx, cancel := signal.NotifyContext(ctx, sig)
	defer cancel()

	// errgroup.WithContext gives us a derived context (gCtx) that is
	// cancelled when either:
	//   (a) the parent ctx cancels (signal received, test cancels, etc.), or
	//   (b) any goroutine registered via g.Go returns a non-nil error.
	g, gCtx := errgroup.WithContext(ctx)

	// cfg.RequestTimeout is NOT mapped to http.Server.ReadTimeout
	// here — ReadTimeout only governs the time the server waits to
	// read an incoming request (headers + body), not the time the
	// handler is allowed to run. This template originally conflated
	// the two because the flag is called --request-timeout, but the
	// intent (strictly bound handler execution) is satisfied by
	// http.TimeoutHandler wrapping inside NewServer. See server.go
	// for the handler graph layering.
	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      NewServer(cfg),
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		// BaseContext is called once by Serve with the bound listener,
		// right before it starts accepting connections. We piggy-back on
		// it to surface the resolved address to the ready callback —
		// useful for tests that bind on port 0 and need to discover the
		// port the kernel picked. Returning gCtx ties in-flight request
		// contexts to the overall lifecycle, so they cancel on shutdown.
		BaseContext: func(ln net.Listener) context.Context {
			if ready != nil {
				ready(ln.Addr().String())
			}
			return gCtx
		},
	}

	// Serve goroutine. http.ErrServerClosed is the expected sentinel
	// returned after a successful Shutdown and is not a real error.
	//
	// The startup banner is written directly to out via fmt rather than
	// slog so it always prints regardless of LOG_LEVEL — the default
	// level is "error" and we still want operators to see what version
	// of the binary started and where it's listening.
	g.Go(func() error {
		fmt.Fprintf(out, "starting server addr=%s version=%s commit=%s date=%s\n",
			httpServer.Addr, version, commit, date)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	// Shutdown watcher. Blocks until gCtx cancels (signal, parent
	// cancel, or a sibling goroutine errored), then drains the HTTP
	// server and flushes metrics sequentially with INDEPENDENT
	// bounded contexts.
	//
	// Independent contexts matter: a busy HTTP shutdown can burn the
	// entire cfg.ShutdownTimeout waiting on in-flight requests, and
	// if metrics shutdown shared the same context it would find it
	// already expired and drop pending exports on the floor. Each
	// phase gets its own fresh deadline so both have a fair chance
	// to do their job. Worst-case wall time is 2*ShutdownTimeout,
	// which is acceptable in exchange for not losing data.
	//
	// Sequential (not concurrent) ordering is deliberate: the HTTP
	// server drains its in-flight requests first, which can emit
	// final metrics via otelhttp; running metrics shutdown after
	// ensures those final records are captured by the provider
	// before its exporters tear down. The contexts are derived from
	// context.Background rather than gCtx because gCtx is already
	// cancelled at this point.
	//
	// Both errors are joined so a failure in one does not hide a
	// failure in the other.
	g.Go(func() error {
		<-gCtx.Done()
		slog.InfoContext(ctx, "Server shutting down")

		httpCtx, httpCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		httpErr := httpServer.Shutdown(httpCtx)
		httpCancel()
		if httpErr != nil {
			httpErr = fmt.Errorf("http shutdown: %w", httpErr)
		}

		metricsCtx, metricsCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		metricsErr := metricsShutdown(metricsCtx)
		metricsCancel()
		if metricsErr != nil {
			metricsErr = fmt.Errorf("metrics shutdown: %w", metricsErr)
		}

		return errors.Join(httpErr, metricsErr)
	})

	return g.Wait()
}
