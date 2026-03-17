package handle

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/mikecsmith/httplab/internal/respond"
)

type EchoResponse struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Proto   string              `json:"proto"`
	Headers map[string][]string `json:"headers"`
	Body    json.RawMessage     `json:"body,omitempty"`
}

// EchoGet returns a handler that reflects the inbound GET request
// back to the caller.
func EchoGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e := EchoResponse{
			Method:  r.Method,
			URL:     r.URL.String(),
			Proto:   r.Proto,
			Headers: r.Header.Clone(),
		}

		respond.With(w, r, 200, e)
	}
}

// EchoPost returns a handler that reflects the inbound POST request
// back to the caller, including the request body.
func EchoPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			slog.ErrorContext(r.Context(), "Unable to read body content", "error", err)
			respond.WithError(w, r, respond.ErrBadRequest.WithDetails("Unable to parse body contents"))
			return
		}
		e := EchoResponse{
			Method:  r.Method,
			URL:     r.URL.String(),
			Proto:   r.Proto,
			Headers: r.Header.Clone(),
		}

		if r.Header.Get("Content-Type") == "application/json" && json.Valid(b) {
			e.Body = b
		} else {
			e.Body, _ = json.Marshal(string(b))
		}

		slog.InfoContext(r.Context(), "EchoPost called")

		respond.With(w, r, http.StatusOK, e)
	}
}
