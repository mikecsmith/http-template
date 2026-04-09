package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mikecsmith/httplab/internal/logger"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, w io.Writer) error {
	portPtr := flag.Int("port", 8080, "The port used by the HTTP server")
	flag.Parse()

	logger.Init(w)

	addr := fmt.Sprintf(":%d", *portPtr)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	mux := http.NewServeMux()
	addRoutes(mux)

	srvErr := make(chan error, 1)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.InfoContext(ctx, "Starting server", "addr", srv.Addr)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		slog.InfoContext(ctx, "Server shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return srv.Shutdown(shutdownCtx)
}
