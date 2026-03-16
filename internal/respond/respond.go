// Package respond
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// ResponseError is the JSON shape written to the client on any error.
type ResponseError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Details string `json:"details,omitempty"`
}

func (re ResponseError) WithDetails(details string) ResponseError {
	re.Details = details
	return re
}

// SuccessResponse is the standard envelope for successful responses with any status code.
// Data holds the handler-specific payload.
type SuccessResponse struct {
	Data any `json:"data"`
}

// WithOK writes a 200 OK response with the provided data encoded as JSON.
func WithOK(w http.ResponseWriter, r *http.Request, data any) {
	With(w, r, http.StatusOK, SuccessResponse{Data: data})
}

// WithError accepts a ResponseError and passes it to With
func WithError(w http.ResponseWriter, r *http.Request, re ResponseError) {
	With(w, r, re.Code, re)
}

// With writes a response with the given status code and data as JSON.
func With(w http.ResponseWriter, r *http.Request, status int, data any) {
	ctx := r.Context()

	b, err := json.Marshal(data)
	if err != nil {
		if typeErr, ok := errors.AsType[*json.UnsupportedTypeError](err); ok {
			slog.ErrorContext(ctx, "Attempted to marshal an unsupported type during an API response", "type", typeErr.Type.String(), "error", typeErr)
		} else {
			slog.ErrorContext(ctx, "An unknown error occurred marshalling an API response", "error", err)
		}
		WithError(w, r, ErrInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(b); err != nil {
		slog.ErrorContext(ctx, "Failed to write response", "error", err)
	}
}

var (
	ErrInternalServerError = ResponseError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrNotFound            = ResponseError{Code: http.StatusNotFound, Message: "not found"}
	ErrBadRequest          = ResponseError{Code: http.StatusBadRequest, Message: "bad request"}
)
