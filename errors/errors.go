// Package errors defines sanitized, transport-neutral API errors.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct { Status int; Code string; Message string; cause error }
func (e *Error) Error() string { return e.Code + ": " + e.Message }
func (e *Error) Unwrap() error { return e.cause }
func New(status int, code, message string) *Error { return &Error{Status: status, Code: code, Message: message} }
func Wrap(err *Error, cause error) *Error { return &Error{Status: err.Status, Code: err.Code, Message: err.Message, cause: cause} }
func Cause(err *Error) error { if err == nil { return nil }; return err.cause }
func Map(err error) *Error { if err == nil { return nil }; var apiErr *Error; if errors.As(err, &apiErr) { return apiErr }; return Wrap(Internal(), err) }
func Validation(field, reason string) *Error { return New(http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("field %q %s", field, reason)) }
func BadRequest(message string) *Error { return New(http.StatusBadRequest, "BAD_REQUEST", message) }
func NotFound(resource string) *Error { return New(http.StatusNotFound, "NOT_FOUND", resource+" not found") }
func Conflict(message string) *Error { return New(http.StatusConflict, "CONFLICT", message) }
func Unauthorized() *Error { return New(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required") }
func Forbidden() *Error { return New(http.StatusForbidden, "FORBIDDEN", "access denied") }
func TooManyRequests() *Error { return New(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests") }
func ServiceUnavailable() *Error { return New(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service temporarily unavailable") }
func Internal() *Error { return New(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error") }
