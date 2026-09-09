// Package errors provides typed HTTP errors for the API layer.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is a typed HTTP error carrying a status code, machine-readable code
// and message, with optional support for wrapping a cause.
type Error struct {
	Status  int
	Code    string
	Message string
	cause   error
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Unwrap returns the wrapped cause, if any.
func (e *Error) Unwrap() error { return e.cause }

// New returns a new Error with the given status, code and message.
func New(s int, c, m string) *Error { return &Error{Status: s, Code: c, Message: m} }

// Wrap returns a new Error with the same status, code and message as e,
// wrapping the given cause.
func Wrap(e *Error, c error) *Error {
	return &Error{Status: e.Status, Code: e.Code, Message: e.Message, cause: c}
}

// Cause returns the wrapped cause of e, or nil when e is nil or has no cause.
func Cause(e *Error) error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Map converts any error into an *Error. nil stays nil; an existing *Error is
// returned as-is; anything else is wrapped as an Internal error.
func Map(e error) *Error {
	if e == nil {
		return nil
	}
	var a *Error
	if errors.As(e, &a) {
		return a
	}
	return Wrap(Internal(), e)
}

// Validation returns a 400 Bad Request Error for the given field and reason.
func Validation(f, r string) *Error {
	return New(http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("field %q %s", f, r))
}

// Internal returns a 500 Internal Server Error.
func Internal() *Error {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
