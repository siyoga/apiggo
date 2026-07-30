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

func (e *Error) Error() string   { return e.Message }
func (e *Error) StatusCode() int { return e.Status }
func (e *Error) Body() any       { return e }

func ErrBadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Message: msg}
}

func ErrForbidden(msg string) *Error {
	return &Error{Status: http.StatusForbidden, Message: msg}
}

func ErrNotFound(msg string) *Error {
	return &Error{Status: http.StatusNotFound, Message: msg}
}

func ErrInternal(msg string) *Error {
	return &Error{Status: http.StatusInternalServerError, Message: msg}
}
