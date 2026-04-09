package respond_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mikecsmith/httplab/internal/respond"
)

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodGet, "/", nil)
}

func TestWithOK(t *testing.T) {
	t.Run("sets status 200 and application/json content type", func(t *testing.T) {
		w := httptest.NewRecorder()
		respond.WithOK(w, newRequest(t), map[string]string{"hello": "world"})

		if w.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
		}

		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("got Content-Type %q, want %q", ct, "application/json")
		}
	})

	t.Run("writes data directly as JSON body without envelope", func(t *testing.T) {
		w := httptest.NewRecorder()
		respond.WithOK(w, newRequest(t), map[string]string{"hello": "world"})

		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if body["hello"] != "world" {
			t.Errorf("got %q, want %q", body["hello"], "world")
		}
		if _, exists := body["data"]; exists {
			t.Error("expected no envelope wrapper, but found 'data' key")
		}
	})

	t.Run("returns error and writes 500 for unencodable type", func(t *testing.T) {
		w := httptest.NewRecorder()
		respond.WithOK(w, newRequest(t), make(chan int))

		if w.Code != http.StatusInternalServerError {
			t.Errorf("got status %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name       string
		re         respond.ResponseError
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "not found",
			re:         respond.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "not found",
		},
		{
			name:       "bad request",
			re:         respond.ErrBadRequest,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "bad request",
		},
		{
			name:       "internal server error",
			re:         respond.ErrInternalServerError,
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "internal server error",
		},
		{
			name:       "unprocessable entity",
			re:         respond.ErrUnprocessableEntity,
			wantStatus: http.StatusUnprocessableEntity,
			wantMsg:    "unprocessable entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respond.WithError(w, newRequest(t), tt.re)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}
			if body["error"] != tt.wantMsg {
				t.Errorf("got error %q, want %q", body["error"], tt.wantMsg)
			}
		})
	}
}

func TestWithErrorMessage(t *testing.T) {
	t.Run("overrides the default error message", func(t *testing.T) {
		w := httptest.NewRecorder()
		message := "meter reading 123 does not exist"
		respond.WithError(w, newRequest(t), respond.ErrNotFound.WithErrorMessage(message))

		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if body["error"] != message {
			t.Errorf("got error %q, want %q", body["error"], message)
		}
	})

	t.Run("does not mutate the sentinel", func(t *testing.T) {
		respond.ErrNotFound.WithErrorMessage("something else")

		w := httptest.NewRecorder()
		respond.WithError(w, newRequest(t), respond.ErrNotFound)

		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if body["error"] != "not found" {
			t.Errorf("sentinel was mutated: got error %q, want %q", body["error"], "not found")
		}
	})
}

func TestWithDetails(t *testing.T) {
	t.Run("includes details map in response", func(t *testing.T) {
		w := httptest.NewRecorder()
		details := map[string]string{"name": "is required", "email": "is required"}
		respond.WithError(w, newRequest(t), respond.ErrUnprocessableEntity.WithDetails(details))

		var body struct {
			Error   string            `json:"error"`
			Details map[string]string `json:"details"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if body.Details["name"] != "is required" {
			t.Errorf("got name detail %q, want %q", body.Details["name"], "is required")
		}
		if body.Details["email"] != "is required" {
			t.Errorf("got email detail %q, want %q", body.Details["email"], "is required")
		}
	})

	t.Run("omits details field when empty", func(t *testing.T) {
		w := httptest.NewRecorder()
		respond.WithError(w, newRequest(t), respond.ErrNotFound)

		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if _, exists := body["details"]; exists {
			t.Error("expected details to be omitted when empty, but key was present")
		}
	})

	t.Run("does not mutate the sentinel", func(t *testing.T) {
		respond.ErrBadRequest.WithDetails(map[string]string{"field": "problem"})

		w := httptest.NewRecorder()
		respond.WithError(w, newRequest(t), respond.ErrBadRequest)

		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if _, exists := body["details"]; exists {
			t.Error("sentinel was mutated: details should be omitted")
		}
	})
}

func TestWith(t *testing.T) {
	t.Run("custom status code is respected, application/json content type", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := "I'm a little teapot, short and stout"

		respond.With(w, newRequest(t), http.StatusTeapot, payload)

		var result string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if result != payload {
			t.Errorf("got %q, want %q", result, payload)
		}

		if w.Code != http.StatusTeapot {
			t.Errorf("got status %d, want %d", w.Code, http.StatusTeapot)
		}

		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("got Content-Type %q, want %q", ct, "application/json")
		}
	})
}
