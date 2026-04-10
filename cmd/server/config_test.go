package main

import (
	"testing"
)

func TestParseConfig(t *testing.T) {
	noEnv := func(string) string { return "" }

	tests := []struct {
		name     string
		args     []string
		getenv   func(string) string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{
			name:     "defaults when no flags or env",
			args:     []string{"server"},
			getenv:   noEnv,
			wantHost: "",
			wantPort: "8080",
		},
		{
			name:     "flags override defaults",
			args:     []string{"server", "--port", "3000", "--host", "localhost"},
			getenv:   noEnv,
			wantHost: "localhost",
			wantPort: "3000",
		},
		{
			name:   "env vars override flag defaults",
			args:   []string{"server"},
			getenv: func(key string) string {
				switch key {
				case "PORT":
					return "9090"
				case "HOST":
					return "0.0.0.0"
				default:
					return ""
				}
			},
			wantHost: "0.0.0.0",
			wantPort: "9090",
		},
		{
			name:    "invalid flag returns error",
			args:    []string{"server", "--unknown"},
			getenv:  noEnv,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseConfig(tt.args, tt.getenv)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Host != tt.wantHost {
				t.Errorf("got host %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("got port %q, want %q", cfg.Port, tt.wantPort)
			}
		})
	}
}
