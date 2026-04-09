// Package respond provides helpers for writing JSON HTTP responses.
//
// Success responses write the data directly as the JSON body — no envelope
// wrapper. Error responses use a consistent [ResponseError] shape with an
// "error" field and an optional "details" field for request-specific context.
//
// All responses are serialised with [json.Marshal] before any bytes are written
// to the [http.ResponseWriter]. This means that if marshalling fails the status
// code has not yet been sent, so the caller can still receive a clean 500 error
// instead of a partial or malformed response.
//
// Sentinel errors ([ErrBadRequest], [ErrNotFound], [ErrInternalServerError])
// cover the most common failure cases and can be extended with
// [ResponseError.WithDetails] to add request-specific information without
// mutating the original sentinel.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// ResponseError is the JSON shape written to the client on any error.
// Code is used to set the HTTP status but is excluded from the JSON body.
// Message is a stable, client-safe description of the error category.
// Details is an optional field for request-specific context.
type ResponseError struct {
	Code         int               `json:"-"`
	ErrorMessage string            `json:"error"`
	Details      map[string]string `json:"details,omitempty"`
}

// WithMessage allows overwriting the default message on copies of sentinel errors
func (re ResponseError) WithErrorMessage(message string) ResponseError {
	re.ErrorMessage = message
	return re
}

// WithDetails returns a copy of the error with the Details field set.
// The receiver is not mutated, so sentinel errors remain safe for reuse.
// Details is typically the problems map returned by [request.Validator.Valid].
func (re ResponseError) WithDetails(details map[string]string) ResponseError {
	re.Details = details
	return re
}

// WithOK writes a 200 OK response with data serialised directly as the JSON body.
func WithOK(w http.ResponseWriter, r *http.Request, data any) {
	With(w, r, http.StatusOK, data)
}

// WithError writes an error response using the status code from re.
func WithError(w http.ResponseWriter, r *http.Request, re ResponseError) {
	With(w, r, re.Code, re)
}

// With writes a JSON response with the given status code. Data is marshalled
// before writing so that encoding errors can be caught and returned as a 500
// without sending a partial response to the client.
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
		slog.WarnContext(ctx, "Failed to write response", "error", err)
	}
}

// Sentinel errors for common HTTP failure responses.
// Use [ResponseError.WithDetails] to add request-specific context.
var (
	ErrInternalServerError = ResponseError{Code: http.StatusInternalServerError, ErrorMessage: "internal server error"}
	ErrNotFound            = ResponseError{Code: http.StatusNotFound, ErrorMessage: "not found"}
	ErrBadRequest          = ResponseError{Code: http.StatusBadRequest, ErrorMessage: "bad request"}
	ErrUnprocessableEntity = ResponseError{Code: http.StatusUnprocessableEntity, ErrorMessage: "unprocessable entity"}
)
