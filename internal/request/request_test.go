package request_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mikecsmith/http-template/internal/request"
)

type testPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (v testPayload) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	if v.Name == "" {
		problems["name"] = "is required"
	}
	if v.Email == "" {
		problems["email"] = "is required"
	}
	return problems
}

func newPostRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid JSON body",
			body:    `{"name":"Ada","email":"ada@example.com"}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    `{not json}`,
			wantErr: true,
		},
		{
			name:    "unknown fields rejected",
			body:    `{"name":"Ada","unknown_field":"value"}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    "",
			wantErr: true,
		},
		{
			name:    "body exceeds max size",
			body:    `{"name":"` + strings.Repeat("a", request.MaxBodySize+1) + `"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPostRequest(t, tt.body)

			got, err := request.Decode[testPayload](r)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != "Ada" {
				t.Errorf("got name %q, want %q", got.Name, "Ada")
			}
			if got.Email != "ada@example.com" {
				t.Errorf("got email %q, want %q", got.Email, "ada@example.com")
			}
		})
	}
}

func TestDecodeValid(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantErr      bool
		wantProblems int
	}{
		{
			name:         "valid data passes validation",
			body:         `{"name":"Ada","email":"ada@example.com"}`,
			wantErr:      false,
			wantProblems: 0,
		},
		{
			name:         "empty fields fail validation",
			body:         `{"name":"","email":""}`,
			wantErr:      false,
			wantProblems: 2,
		},
		{
			name:         "malformed JSON returns error before validation",
			body:         `{not json}`,
			wantErr:      true,
			wantProblems: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPostRequest(t, tt.body)

			got, problems, err := request.DecodeValid[testPayload](r)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(problems) != tt.wantProblems {
				t.Fatalf("got %d problems, want %d: %v", len(problems), tt.wantProblems, problems)
			}
			if tt.wantProblems == 0 && got.Name != "Ada" {
				t.Errorf("got name %q, want %q", got.Name, "Ada")
			}
		})
	}
}
