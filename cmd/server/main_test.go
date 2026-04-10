package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestRun_E2E boots the real server via run() on an ephemeral port,
// waits for it to be ready, exercises every route via a table of cases,
// and then triggers a graceful shutdown by cancelling the context.
//
// This single test covers routing, middleware, decode/validate/respond,
// and lifecycle.
func TestRun_E2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	baseURL, errCh := startServer(t, ctx)
	waitForReady(t, ctx, baseURL+"/readyz")

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		checks []check
	}{
		{
			name:   "GET /healthz returns 200 without request id but with security headers",
			method: http.MethodGet,
			path:   "/healthz",
			checks: []check{
				hasStatus(http.StatusOK),
				headerAbsent("X-Request-ID"),
				headerEquals("X-Content-Type-Options", "nosniff"),
				headerEquals("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'"),
			},
		},
		{
			name:   "GET /readyz returns 200 without request id",
			method: http.MethodGet,
			path:   "/readyz",
			checks: []check{
				hasStatus(http.StatusOK),
				headerAbsent("X-Request-ID"),
			},
		},
		{
			name:   "GET /hello returns 200 with request id",
			method: http.MethodGet,
			path:   "/hello",
			checks: []check{
				hasStatus(http.StatusOK),
				headerPresent("X-Request-ID"),
			},
		},
		{
			name:   "POST /hello with valid body returns 200",
			method: http.MethodPost,
			path:   "/hello",
			body:   `{"name":"Foo"}`,
			checks: []check{
				hasStatus(http.StatusOK),
				headerPresent("X-Request-ID"),
				bodyMessageContains("Foo"),
			},
		},
		{
			name:   "POST /hello with missing name returns 422 with details",
			method: http.MethodPost,
			path:   "/hello",
			body:   `{}`,
			checks: []check{
				hasStatus(http.StatusUnprocessableEntity),
				bodyHasDetail("name"),
			},
		},
		{
			name:   "POST /hello with malformed JSON returns 400",
			method: http.MethodPost,
			path:   "/hello",
			body:   `not json`,
			checks: []check{
				hasStatus(http.StatusBadRequest),
			},
		},
		{
			name:   "POST /hello with unknown field returns 400",
			method: http.MethodPost,
			path:   "/hello",
			body:   `{"name":"Mike","extra":1}`,
			checks: []check{
				hasStatus(http.StatusBadRequest),
			},
		},
		{
			name:   "unknown path returns 404 JSON",
			method: http.MethodGet,
			path:   "/does-not-exist",
			checks: []check{
				hasStatus(http.StatusNotFound),
				contentTypeContains("application/json"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := doRequest(t, ctx, tt.method, baseURL+tt.path, tt.body)
			for _, c := range tt.checks {
				c(t, resp, body)
			}
		})
	}

	// Trigger graceful shutdown by cancelling the parent context. Because
	// run() wraps ctx with errgroup.WithContext, cancelling here propagates
	// through gCtx to both the serve goroutine and the shutdown watcher.
	// g.Wait() then returns whatever error (if any) the watchers produced.
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("run returned error on shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not exit within shutdown timeout")
	}
}

// startServer boots run() on an ephemeral port in a goroutine, waits for
// the ready callback to fire, and returns the base URL and the channel
// on which run() will eventually publish its exit error. The caller is
// expected to cancel ctx and drain errCh to shut the server down.
func startServer(t *testing.T, ctx context.Context) (baseURL string, errCh <-chan error) {
	t.Helper()

	addrCh := make(chan string, 1)
	errs := make(chan error, 1)

	args := []string{
		"server",
		"--host", "127.0.0.1",
		"--port", "0",
		// Keep shutdown snappy so a broken test doesn't hang for 10s.
		"--shutdown-timeout", "2s",
	}
	getenv := func(string) string { return "" }

	go func() {
		// SIGUSR1 is used instead of os.Interrupt so the test's signal
		// handler can't be tripped by anything external — shutdown is
		// driven entirely by cancelling ctx.
		errs <- run(ctx, args, getenv, io.Discard, syscall.SIGUSR1, func(addr string) {
			addrCh <- addr
		})
	}()

	select {
	case addr := <-addrCh:
		return "http://" + addr, errs
	case err := <-errs:
		t.Fatalf("run exited before ready: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server to bind")
	}
	return "", nil // unreachable
}

// check is one assertion over a response. Checks receive the already-drained
// body as a []byte so multiple checks in the same case can each look at it
// without having to worry about resp.Body being a one-shot reader.
type check func(t *testing.T, resp *http.Response, body []byte)

func hasStatus(want int) check {
	return func(t *testing.T, resp *http.Response, _ []byte) {
		t.Helper()
		if resp.StatusCode != want {
			t.Errorf("status = %d, want %d", resp.StatusCode, want)
		}
	}
}

func headerPresent(name string) check {
	return func(t *testing.T, resp *http.Response, _ []byte) {
		t.Helper()
		if resp.Header.Get(name) == "" {
			t.Errorf("expected %s header to be set", name)
		}
	}
}

func headerAbsent(name string) check {
	return func(t *testing.T, resp *http.Response, _ []byte) {
		t.Helper()
		if got := resp.Header.Get(name); got != "" {
			t.Errorf("%s = %q, want empty", name, got)
		}
	}
}

func headerEquals(name, want string) check {
	return func(t *testing.T, resp *http.Response, _ []byte) {
		t.Helper()
		if got := resp.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func contentTypeContains(want string) check {
	return func(t *testing.T, resp *http.Response, _ []byte) {
		t.Helper()
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, want) {
			t.Errorf("content-type = %q, want it to contain %q", ct, want)
		}
	}
}

func bodyMessageContains(want string) check {
	return func(t *testing.T, _ *http.Response, body []byte) {
		t.Helper()
		got := unmarshalJSON(t, body)
		msg, _ := got["message"].(string)
		if !strings.Contains(msg, want) {
			t.Errorf("message = %q, want it to contain %q", msg, want)
		}
	}
}

func bodyHasDetail(key string) check {
	return func(t *testing.T, _ *http.Response, body []byte) {
		t.Helper()
		got := unmarshalJSON(t, body)
		details, _ := got["details"].(map[string]any)
		if _, ok := details[key]; !ok {
			t.Errorf("expected details.%s in body, got %v", key, got)
		}
	}
}

// waitForReady polls url until it returns 200 OK or the deadline passes.
// It takes testing.TB so it can be called from tests or benchmarks.
func waitForReady(t testing.TB, ctx context.Context, url string) {
	t.Helper()

	client := http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for %s", url)
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("server at %s not ready within deadline", url)
}

// doRequest sends an HTTP request, drains and closes the response body,
// and returns the response together with the body bytes. Callers get to
// inspect status/headers via resp and the payload via body, without
// having to worry about closing the reader. Any transport, read, or
// close failure fails the test via t.Fatal.
func doRequest(t testing.TB, ctx context.Context, method, url, body string) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, respBody
}

// unmarshalJSON parses a response body into a loose map. Intended for
// e2e assertions where the exact shape isn't worth a dedicated type.
func unmarshalJSON(t testing.TB, body []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return got
}
