// Package request provides helpers for decoding incoming HTTP request bodies.
//
// [Decode] handles the common path: read the body up to [MaxBodySize] bytes,
// reject unknown JSON fields, and return a typed value. [DecodeValid] extends
// this by also running the decoded value through the [Validator] interface,
// returning a map of field-level problems when the input is structurally valid
// but semantically incorrect.
package request

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MaxBodySize is the maximum number of bytes read from a request body (1 MB).
const MaxBodySize = 1_048_576

// Validator is implemented by types that can check their own field values.
// Valid returns a map of field names to human-readable problem descriptions.
// An empty or nil map means the value is valid.
type Validator interface {
	Valid(ctx context.Context) (problems map[string]string)
}

// Decode reads the request body as JSON into a value of type T. The body is
// limited to [MaxBodySize] bytes and unknown fields are rejected.
func Decode[T any](r *http.Request) (T, error) {
	var v T

	r.Body = http.MaxBytesReader(nil, r.Body, MaxBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}

	return v, nil
}

// DecodeValid decodes the request body like [Decode] and then calls Valid on
// the result. If validation fails the decoded value and the problems map are
// both returned with a nil error, allowing the caller to decide how to respond.
// A non-nil error means decoding itself failed.
func DecodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	v, err := Decode[T](r)
	if err != nil {
		return v, nil, fmt.Errorf("validator: %w", err)
	}

	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, nil
	}

	return v, nil, nil
}
