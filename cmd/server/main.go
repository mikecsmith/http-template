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

	"golang.org/x/sync/errgroup"

	"github.com/mikecsmith/httplab/internal/config"
	"github.com/mikecsmith/httplab/internal/logger"
	"github.com/mikecsmith/httplab/internal/metrics"
)

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

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      NewServer(cfg),
		ReadTimeout:  cfg.RequestTimeout,
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
	g.Go(func() error {
		slog.InfoContext(gCtx, "Starting server", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	// Shutdown watcher. Blocks until gCtx cancels (signal, parent cancel,
	// or a sibling goroutine errored), then calls Shutdown with a bounded
	// timeout. The shutdown context is derived from context.Background
	// rather than gCtx because gCtx is already cancelled at this point —
	// we need a fresh deadline to bound the shutdown itself. The same
	// bounded context is used to flush and tear down the MeterProvider
	// before returning, so pending metric records get one last push at
	// any exporter that was configured.
	g.Go(func() error {
		<-gCtx.Done()
		slog.InfoContext(ctx, "Server shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		if err := metricsShutdown(shutdownCtx); err != nil {
			return fmt.Errorf("metrics shutdown: %w", err)
		}
		return nil
	})

	return g.Wait()
}
