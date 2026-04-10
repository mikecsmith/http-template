// Package config contains the runtime configuration used throughout the API
package config

import (
	"flag"
	"fmt"
)

// Config holds the server's runtime configuration. Fields are strings
// because [net.JoinHostPort] expects them that way. No validation is
// performed here — invalid values (e.g. a non-numeric port) will
// surface as a clear error from [net.Listen] at startup.
type Config struct {
	Host string
	Port string
}

// ParseConfig builds a [config] from command-line flags and environment
// variables. Flags are parsed first using a local [flag.FlagSet] (not
// the global one) so that parallel test calls to [run] don't interfere.
// Environment variables override flag values when set.
func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	var cfg Config
	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	flags.StringVar(&cfg.Port, "port", "8080", "The port used by the HTTP server")
	flags.StringVar(&cfg.Host, "host", "", "The host used by the HTTP server")

	if err := flags.Parse(args[1:]); err != nil {
		return cfg, fmt.Errorf("ParseConfig flags: %w", err)
	}

	if v := getenv("PORT"); v != "" {
		cfg.Port = v
	}

	if v := getenv("HOST"); v != "" {
		cfg.Host = v
	}

	return cfg, nil
}
