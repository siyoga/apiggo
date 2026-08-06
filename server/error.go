// Package server is the thin net/http runtime that hosts apiggo-generated code:
// option-based configuration, a middleware chain, panic recovery, unified error
// handling via the APIError interface, and graceful shutdown.
package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorHandler turns any error returned by a handler (or a recovered panic
// converted to an error) into an HTTP response. It owns the APIError typecast
// and the unknown-error -> 500 fallback, so the adapter and panicMiddleware render
// errors through one place instead of duplicating the mapping.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// defaultErrorHandler maps err to an APIError (falling back to 500) and
// serializes its body as JSON with the status code.
func (o *serverOptions) defaultErrorHandler() {
	o.errorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		var ae APIError
		if !errors.As(err, &ae) {
			ae = ErrInternal("internal server error")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ae.StatusCode())
		_ = json.NewEncoder(w).Encode(ae.Body())
	}
}

// APIError is returned from handlers as a normal error; the runtime maps it to
// an HTTP status and response body via errors.As.
type APIError interface {
	error
	StatusCode() int
	Body() any
}

// Error is the ready-made APIError implementation for the common case. Since
// APIError.Body returns any, callers may also return their own type generated
// from the contract's error schema.
type Error struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// StatusCode reports the HTTP status this error maps to.
func (e *Error) StatusCode() int { return e.Status }

// Body returns the value serialized as the JSON response body.
func (e *Error) Body() any { return e }

// ErrBadRequest returns a 400 Error with the given message.
func ErrBadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: msg}
}

// ErrForbidden returns a 403 Error with the given message.
func ErrForbidden(msg string) *Error {
	return &Error{Status: http.StatusForbidden, Message: msg}
}

// ErrNotFound returns a 404 Error with the given message.
func ErrNotFound(msg string) *Error {
	return &Error{Status: http.StatusNotFound, Message: msg}
}

// ErrInternal returns a 500 Error with the given message.
func ErrInternal(msg string) *Error {
	return &Error{Status: http.StatusInternalServerError, Message: msg}
}
