// Package main is the entry point for the HTTP server.
//
// main is kept deliberately minimal — it calls [run] with the real OS
// dependencies and translates any error into a non-zero exit code. All
// startup logic lives in [run] so the entire program can be exercised
// from tests by calling run directly with controlled inputs.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mikecsmith/httplab/internal/logger"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdout, os.Interrupt); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// run is the real entry point. It takes OS fundamentals as arguments so
// that tests can call it with controlled values — fake args, env vars,
// and output writers — without mutating global state. The sig parameter
// controls which OS signal triggers graceful shutdown.
func run(ctx context.Context, args []string, getenv func(string) string, out io.Writer, sig os.Signal) error {
	logger.Init(out)

	cfg, err := parseConfig(args, getenv)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, sig)
	defer cancel()

	mux := http.NewServeMux()
	addRoutes(mux, cfg)

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	httpServerErr := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "Starting server", "addr", httpServer.Addr)
		httpServerErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-httpServerErr:
		return err
	case <-ctx.Done():
		slog.InfoContext(ctx, "Server shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return httpServer.Shutdown(shutdownCtx)
}
