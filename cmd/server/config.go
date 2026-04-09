package main

import (
	"flag"
	"fmt"
)

type config struct {
	Host string
	Port string
}

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
