package main

import (
	"flag"
	"fmt"
)

// config holds the server's runtime configuration. Fields are strings
// because [net.JoinHostPort] expects them that way. No validation is
// performed here — invalid values (e.g. a non-numeric port) will
// surface as a clear error from [net.Listen] at startup.
type config struct {
	Host string
	Port string
}

// parseConfig builds a [config] from command-line flags and environment
// variables. Flags are parsed first using a local [flag.FlagSet] (not
// the global one) so that parallel test calls to [run] don't interfere.
// Environment variables override flag values when set, following the
// convention that deployment config (env) takes precedence over defaults
// (flags) but explicit flags override both.
func parseConfig(args []string, getenv func(string) string) (config, error) {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	port := flags.String("port", "8080", "The port used by the HTTP server")
	host := flags.String("host", "", "The host used by the HTTP server")

	if err := flags.Parse(args[1:]); err != nil {
		return config{}, fmt.Errorf("parseConfig flags: %w", err)
	}

	if v := getenv("PORT"); v != "" {
		*port = v
	}

	if v := getenv("HOST"); v != "" {
		*host = v
	}

	cfg := config{
		Host: *host,
		Port: *port,
	}

	return cfg, nil
}
