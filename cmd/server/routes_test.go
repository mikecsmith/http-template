package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mikecsmith/httplab/internal/config"
	"github.com/mikecsmith/httplab/internal/middleware"
)

func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	addRoutes(mux, config.Config{})
	return mux
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "GET /healthz returns 200",
			method:     http.MethodGet,
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET /readyz returns 200",
			method:     http.MethodGet,
			path:       "/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET /hello returns 200",
			method:     http.MethodGet,
			path:       "/hello",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /hello with valid body returns 200",
			method:     http.MethodPost,
			path:       "/hello",
			body:       `{"name":"Ada"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /hello with empty name returns 422",
			method:     http.MethodPost,
			path:       "/hello",
			body:       `{"name":""}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "POST /hello with malformed JSON returns 400",
			method:     http.MethodPost,
			path:       "/hello",
			body:       `{not json}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown path returns 404",
			method:     http.MethodGet,
			path:       "/nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrong method on /hello falls through to catch-all 404",
			method:     http.MethodDelete,
			path:       "/hello",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newMux(t)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRoutesNotFoundReturnsJSON(t *testing.T) {
	mux := newMux(t)
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("got Content-Type %q, want %q", ct, "application/json")
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if body["error"] != "not found" {
		t.Errorf("got error %q, want %q", body["error"], "not found")
	}
}

func TestHealthRoutesHaveNoRequestID(t *testing.T) {
	mux := newMux(t)

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if got := w.Header().Get(middleware.RequestIDHeader); got != "" {
				t.Errorf("health probe %s should not have %s header, got %q", path, middleware.RequestIDHeader, got)
			}
		})
	}
}

func TestAPIRoutesHaveRequestID(t *testing.T) {
	mux := newMux(t)
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	got := w.Header().Get(middleware.RequestIDHeader)
	if got == "" {
		t.Error("API route /hello should have X-Request-ID header, got empty")
	}
	if len(got) != 32 {
		t.Errorf("expected 32-char request ID, got %d chars: %s", len(got), got)
	}
}
