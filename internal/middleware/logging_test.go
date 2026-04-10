package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/mikecsmith/httplab/internal/middleware"
)

// Aliases to the package constants so each subtest reads cleanly.
const (
	attrStatus   = middleware.AttrStatus
	attrBytes    = middleware.AttrBytes
	attrDuration = middleware.AttrDuration
)

// captureLogs swaps slog.Default() with a JSON handler writing into a
// bytes.Buffer and returns the buffer plus a cleanup func that restores
// the previous default logger. Tests use it to assert on the record
// emitted by the Logging middleware.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// decodeRecord parses the single-line JSON record the handler wrote and
// fails the test if the buffer doesn't hold exactly one record.
func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	if buf.Len() == 0 {
		t.Fatal("expected a log record, got none")
	}
	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &rec); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	return rec
}

func TestLogging(t *testing.T) {
	// A handler that writes a 200 body should produce a log record with
	// status=200, bytes=len(body), a non-empty duration, and Info level.
	t.Run("captures status, bytes, and duration for 200", func(t *testing.T) {
		buf := captureLogs(t)

		h := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hello"))
		}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		rec := decodeRecord(t, buf)
		if got := rec[attrStatus]; got != float64(200) {
			t.Errorf("status = %v, want 200", got)
		}
		if got := rec[attrBytes]; got != float64(5) {
			t.Errorf("bytes = %v, want 5", got)
		}
		if _, ok := rec[attrDuration]; !ok {
			t.Errorf("duration attr missing from record")
		}
		if got := rec["level"]; got != "INFO" {
			t.Errorf("level = %v, want INFO", got)
		}
		if got := rec["msg"]; got != "request" {
			t.Errorf("msg = %v, want request", got)
		}
	})

	// If the handler writes nothing and never calls WriteHeader, net/http
	// sends an implicit 200 on the wire. The recorder must report the
	// same, so downstream logs match the observable response.
	t.Run("defaults to 200 when handler writes nothing", func(t *testing.T) {
		buf := captureLogs(t)

		h := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		rec := decodeRecord(t, buf)
		if got := rec[attrStatus]; got != float64(200) {
			t.Errorf("status = %v, want 200", got)
		}
		if got := rec[attrBytes]; got != float64(0) {
			t.Errorf("bytes = %v, want 0", got)
		}
	})

	// Level adapts to status: 4xx is Warn, 5xx is Error. Covers the
	// branching in levelForStatus via the public middleware surface.
	t.Run("level adapts to status", func(t *testing.T) {
		cases := []struct {
			name   string
			status int
			level  string
		}{
			{"2xx is info", http.StatusOK, "INFO"},
			{"3xx is info", http.StatusFound, "INFO"},
			{"4xx is warn", http.StatusNotFound, "WARN"},
			{"5xx is error", http.StatusInternalServerError, "ERROR"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				buf := captureLogs(t)

				h := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.status)
				}))
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

				rec := decodeRecord(t, buf)
				if got := rec[attrStatus]; got != float64(tc.status) {
					t.Errorf("status = %v, want %d", got, tc.status)
				}
				if got := rec["level"]; got != tc.level {
					t.Errorf("level = %v, want %s", got, tc.level)
				}
			})
		}
	})

	// Only the first WriteHeader call is recorded — matches net/http's
	// behaviour of ignoring subsequent WriteHeader calls.
	t.Run("records only the first WriteHeader call", func(t *testing.T) {
		buf := captureLogs(t)

		h := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		rec := decodeRecord(t, buf)
		if got := rec[attrStatus]; got != float64(http.StatusTeapot) {
			t.Errorf("status = %v, want %d", got, http.StatusTeapot)
		}
	})
}
