// Package middleware provides HTTP middleware for the server.
//
// Each middleware lives in its own file alongside any unexported
// helpers and exported constants it relies on. See:
//
//   - request_context.go — RequestContext (request ID + logger attrs)
//   - secure_headers.go  — SecureHeaders (baseline security headers)
package middleware
