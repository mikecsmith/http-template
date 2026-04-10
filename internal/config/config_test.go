package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/mikecsmith/httplab/internal/config"
)

func TestParseConfig(t *testing.T) {
	noEnv := func(string) string { return "" }

	tests := []struct {
		name                string
		args                []string
		getenv              func(string) string
		wantHost            string
		wantPort            string
		wantRequestTimeout  time.Duration
		wantWriteTimeout    time.Duration
		wantIdleTimeout     time.Duration
		wantShutdownTimeout time.Duration
		wantLogLevel        slog.Level
		wantErr             bool
	}{
		{
			name:                "defaults when no flags or env",
			args:                []string{"server"},
			getenv:              noEnv,
			wantHost:            config.DefaultHost,
			wantPort:            config.DefaultPort,
			wantRequestTimeout:  config.DefaultRequestTimeout,
			wantWriteTimeout:    config.DefaultWriteTimeout,
			wantIdleTimeout:     config.DefaultIdleTimeout,
			wantShutdownTimeout: config.DefaultShutdownTimeout,
			wantLogLevel:        slog.LevelError,
		},
		{
			name: "flags override defaults",
			args: []string{
				"server",
				"--port", "3000",
				"--host", "localhost",
				"--request-timeout", "1s",
				"--write-timeout", "2s",
				"--idle-timeout", "3s",
				"--shutdown-timeout", "4s",
				"--log-level", "info",
			},
			getenv:              noEnv,
			wantHost:            "localhost",
			wantPort:            "3000",
			wantRequestTimeout:  1 * time.Second,
			wantWriteTimeout:    2 * time.Second,
			wantIdleTimeout:     3 * time.Second,
			wantShutdownTimeout: 4 * time.Second,
			wantLogLevel:        slog.LevelInfo,
		},
		{
			name: "env vars override flag defaults",
			args: []string{"server"},
			getenv: func(key string) string {
				switch key {
				case "PORT":
					return "9091"
				case "HOST":
					return "0.0.0.0"
				case "REQUEST_TIMEOUT":
					return "15s"
				case "WRITE_TIMEOUT":
					return "30s"
				case "IDLE_TIMEOUT":
					return "5m"
				case "SHUTDOWN_TIMEOUT":
					return "15s"
				case "LOG_LEVEL":
					return "warn"
				default:
					return ""
				}
			},
			wantHost:            "0.0.0.0",
			wantPort:            "9091",
			wantRequestTimeout:  15 * time.Second,
			wantWriteTimeout:    30 * time.Second,
			wantIdleTimeout:     5 * time.Minute,
			wantShutdownTimeout: 15 * time.Second,
			wantLogLevel:        slog.LevelWarn,
		},
		{
			name: "env vars override flag values",
			args: []string{
				"server",
				"--request-timeout", "1s",
				"--write-timeout", "2s",
				"--idle-timeout", "3s",
				"--log-level", "info",
			},
			getenv: func(key string) string {
				switch key {
				case "REQUEST_TIMEOUT":
					return "7s"
				case "WRITE_TIMEOUT":
					return "8s"
				case "IDLE_TIMEOUT":
					return "9s"
				case "LOG_LEVEL":
					return "debug"
				default:
					return ""
				}
			},
			wantHost:            config.DefaultHost,
			wantPort:            config.DefaultPort,
			wantRequestTimeout:  7 * time.Second,
			wantWriteTimeout:    8 * time.Second,
			wantIdleTimeout:     9 * time.Second,
			wantShutdownTimeout: config.DefaultShutdownTimeout,
			wantLogLevel:        slog.LevelDebug,
		},
		{
			name:    "invalid flag returns error",
			args:    []string{"server", "--unknown"},
			getenv:  noEnv,
			wantErr: true,
		},
		{
			name: "invalid REQUEST_TIMEOUT env returns error",
			args: []string{"server"},
			getenv: func(key string) string {
				if key == "REQUEST_TIMEOUT" {
					return "not-a-duration"
				}
				return ""
			},
			wantErr: true,
		},
		{
			name: "invalid WRITE_TIMEOUT env returns error",
			args: []string{"server"},
			getenv: func(key string) string {
				if key == "WRITE_TIMEOUT" {
					return "nope"
				}
				return ""
			},
			wantErr: true,
		},
		{
			name: "invalid IDLE_TIMEOUT env returns error",
			args: []string{"server"},
			getenv: func(key string) string {
				if key == "IDLE_TIMEOUT" {
					return "bad"
				}
				return ""
			},
			wantErr: true,
		},
		{
			name: "invalid SHUTDOWN_TIMEOUT env returns error",
			args: []string{"server"},
			getenv: func(key string) string {
				if key == "SHUTDOWN_TIMEOUT" {
					return "bad"
				}
				return ""
			},
			wantErr: true,
		},
		{
			name: "invalid LOG_LEVEL env returns error",
			args: []string{"server"},
			getenv: func(key string) string {
				if key == "LOG_LEVEL" {
					return "banana"
				}
				return ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.args, tt.getenv)
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
			if cfg.RequestTimeout != tt.wantRequestTimeout {
				t.Errorf("got request timeout %v, want %v", cfg.RequestTimeout, tt.wantRequestTimeout)
			}
			if cfg.WriteTimeout != tt.wantWriteTimeout {
				t.Errorf("got write timeout %v, want %v", cfg.WriteTimeout, tt.wantWriteTimeout)
			}
			if cfg.IdleTimeout != tt.wantIdleTimeout {
				t.Errorf("got idle timeout %v, want %v", cfg.IdleTimeout, tt.wantIdleTimeout)
			}
			if cfg.ShutdownTimeout != tt.wantShutdownTimeout {
				t.Errorf("got shutdown timeout %v, want %v", cfg.ShutdownTimeout, tt.wantShutdownTimeout)
			}
			if cfg.LogLevel != tt.wantLogLevel {
				t.Errorf("got log level %v, want %v", cfg.LogLevel, tt.wantLogLevel)
			}
		})
	}
}
