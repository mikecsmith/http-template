// Package config contains the runtime configuration used throughout the API
package config

import (
	"flag"
	"fmt"
	"time"
)

const (
	DefaultPort            = "8080"
	DefaultHost            = ""
	DefaultRequestTimeout  = 5 * time.Second
	DefaultWriteTimeout    = 10 * time.Second
	DefaultIdleTimeout     = 120 * time.Second
	DefaultShutdownTimeout = 10 * time.Second
)

// Config holds the server's runtime configuration. Fields are strings
// because [net.JoinHostPort] expects them that way. No validation is
// performed here — invalid values (e.g. a non-numeric port) will
// surface as a clear error from [net.Listen] at startup.
type Config struct {
	Host            string
	Port            string
	RequestTimeout  time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// ParseConfig builds a [config] from command-line flags and environment
// variables. Flags are parsed first using a local [flag.FlagSet] (not
// the global one) so that parallel test calls to [run] don't interfere.
// Environment variables override flag values when set.
func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	var cfg Config
	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	flags.StringVar(&cfg.Port, "port", DefaultPort, "The port used by the HTTP server")
	flags.StringVar(&cfg.Host, "host", DefaultHost, "The host used by the HTTP server")
	flags.DurationVar(&cfg.RequestTimeout, "request-timeout", DefaultRequestTimeout, "Maximum duration to wait for a request to complete")
	flags.DurationVar(&cfg.WriteTimeout, "write-timeout", DefaultWriteTimeout, "Maximum duration to attempt writing a response")
	flags.DurationVar(&cfg.IdleTimeout, "idle-timeout", DefaultIdleTimeout, "Maximum duration to wait for a request when keep-alive is enabled")
	flags.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", DefaultShutdownTimeout, "Maximum duration to wait before shutting down")

	if err := flags.Parse(args[1:]); err != nil {
		return cfg, fmt.Errorf("ParseConfig flags: %w", err)
	}

	if v := getenv("PORT"); v != "" {
		cfg.Port = v
	}

	if v := getenv("HOST"); v != "" {
		cfg.Host = v
	}

	if err := envToDuration("REQUEST_TIMEOUT", getenv, &cfg.RequestTimeout); err != nil {
		return cfg, err
	}
	if err := envToDuration("WRITE_TIMEOUT", getenv, &cfg.WriteTimeout); err != nil {
		return cfg, err
	}
	if err := envToDuration("IDLE_TIMEOUT", getenv, &cfg.IdleTimeout); err != nil {
		return cfg, err
	}
	if err := envToDuration("SHUTDOWN_TIMEOUT", getenv, &cfg.ShutdownTimeout); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func envToDuration(value string, getenv func(string) string, dur *time.Duration) error {
	if v := getenv(value); v != "" {
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("%s is not valid: %w", value, err)
		}
		*dur = duration
	}
	return nil
}
