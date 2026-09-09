package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is a sanitized API error.
type Error struct {
	Status  int
	Code    string
	Message string
	cause   error
}

// Error returns the public error code and message.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error { return e.cause }

// New creates a sanitized API error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Wrap adds an underlying cause to an API error.
func Wrap(err *Error, cause error) *Error {
	return &Error{Status: err.Status, Code: err.Code, Message: err.Message, cause: cause}
}

// Cause returns the underlying cause.
func Cause(err *Error) error { return err.cause }

// Map converts arbitrary errors to sanitized API errors.
func Map(err error) *Error {
	if err == nil {
		return nil
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return Wrap(Internal(), err)
}

// Validation creates a validation error.
func Validation(field, reason string) *Error {
	return New(http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("field %q %s", field, reason))
}

// BadRequest creates a 400 error.
func BadRequest(message string) *Error { return New(http.StatusBadRequest, "BAD_REQUEST", message) }

// NotFound creates a 404 error.
func NotFound(resource string) *Error {
	return New(http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

// Unauthorized creates a 401 error.
func Unauthorized() *Error {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

// Forbidden creates a 403 error.
func Forbidden() *Error { return New(http.StatusForbidden, "FORBIDDEN", "access denied") }

// Conflict creates a 409 error.
func Conflict(message string) *Error { return New(http.StatusConflict, "CONFLICT", message) }

// TooManyRequests creates a 429 error.
func TooManyRequests() *Error {
	return New(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
}

// ServiceUnavailable creates a 503 error.
func ServiceUnavailable() *Error {
	return New(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service temporarily unavailable")
}

// Internal creates a sanitized 500 error.
func Internal() *Error {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
