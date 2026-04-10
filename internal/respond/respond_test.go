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

func TestWith(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
		wantBody   string
	}{
		{
			name:       "200 with object",
			status:     http.StatusOK,
			data:       map[string]string{"hello": "world"},
			wantStatus: http.StatusOK,
			wantBody:   `{"hello":"world"}`,
		},
		{
			name:       "201 with object",
			status:     http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":"123"}`,
		},
		{
			name:       "custom status code",
			status:     http.StatusTeapot,
			data:       "short and stout",
			wantStatus: http.StatusTeapot,
			wantBody:   `"short and stout"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respond.With(w, newRequest(t), tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("got Content-Type %q, want %q", ct, "application/json")
			}
			got := w.Body.String()
			// json.Marshal appends a newline? No — but let's trim just in case
			if got != tt.wantBody+"\n" && got != tt.wantBody {
				t.Errorf("got body %s, want %s", got, tt.wantBody)
			}
		})
	}

	t.Run("falls back to 500 for unencodable type", func(t *testing.T) {
		w := httptest.NewRecorder()
		respond.With(w, newRequest(t), http.StatusOK, make(chan int))

		if w.Code != http.StatusInternalServerError {
			t.Errorf("got status %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestWithOK(t *testing.T) {
	w := httptest.NewRecorder()
	respond.WithOK(w, newRequest(t), map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWithError(t *testing.T) {
	tests := []struct {
		name       string
		re         respond.ResponseError
		wantStatus int
		wantError  string
	}{
		{
			name:       "bad request",
			re:         respond.ErrBadRequest,
			wantStatus: http.StatusBadRequest,
			wantError:  "bad request",
		},
		{
			name:       "not found",
			re:         respond.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  "not found",
		},
		{
			name:       "internal server error",
			re:         respond.ErrInternalServerError,
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal server error",
		},
		{
			name:       "unprocessable entity",
			re:         respond.ErrUnprocessableEntity,
			wantStatus: http.StatusUnprocessableEntity,
			wantError:  "unprocessable entity",
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
			if body["error"] != tt.wantError {
				t.Errorf("got error %q, want %q", body["error"], tt.wantError)
			}
		})
	}
}

func TestWithErrorMessage(t *testing.T) {
	tests := []struct {
		name      string
		re        respond.ResponseError
		message   string
		wantError string
	}{
		{
			name:      "overrides default message",
			re:        respond.ErrNotFound,
			message:   "meter reading 123 does not exist",
			wantError: "meter reading 123 does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respond.WithError(w, newRequest(t), tt.re.WithErrorMessage(tt.message))

			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}
			if body["error"] != tt.wantError {
				t.Errorf("got error %q, want %q", body["error"], tt.wantError)
			}
		})
	}

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
	tests := []struct {
		name        string
		re          respond.ResponseError
		details     map[string]string
		wantDetails map[string]string
	}{
		{
			name:        "includes details map in response",
			re:          respond.ErrUnprocessableEntity,
			details:     map[string]string{"name": "is required", "email": "is required"},
			wantDetails: map[string]string{"name": "is required", "email": "is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respond.WithError(w, newRequest(t), tt.re.WithDetails(tt.details))

			var body struct {
				Details map[string]string `json:"details"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}
			for k, want := range tt.wantDetails {
				if body.Details[k] != want {
					t.Errorf("got detail %s=%q, want %q", k, body.Details[k], want)
				}
			}
		})
	}

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
