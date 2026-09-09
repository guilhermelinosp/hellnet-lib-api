// Package api defines the core HTTP abstractions used by the modular API core.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Handler processes a request and returns a Response, or an error when the
// handling fails.
type Handler interface {
	Handle(context.Context, Request) (Response, error)
}

// HandlerFunc adapts a plain function to the Handler interface.
type HandlerFunc func(context.Context, Request) (Response, error)

// Handle implements Handler for HandlerFunc.
func (f HandlerFunc) Handle(c context.Context, r Request) (Response, error) {
	return f(c, r)
}

// Request is the read-only view of an incoming HTTP request exposed to
// handlers.
type Request interface {
	Param(string) string
	Query(string) string
	Header(string) string
	Bind(any) error
	Raw() *http.Request
}

// Response is what handlers return; Status and Headers are optional and
// default to 200 and an empty set when zero-valued.
type Response struct {
	Status  int
	Headers http.Header
	Body    any
}

// Route binds an HTTP method and path to a handler, optionally delegating to a
// raw http.Handler.
type Route struct {
	Method  string
	Path    string
	Handler Handler
	Raw     http.Handler
}

// Middleware wraps a Handler, typically to add cross-cutting concerns.
type Middleware func(Handler) Handler

// Router is the composable routing surface of the API core.
type Router interface {
	Handle(string, string, Handler, ...Middleware)
	Mount(string, string, http.Handler)
	Group(string, ...Middleware) Router
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// JSON builds a Response with the given status and body, setting
// Content-Type to application/json.
func JSON(s int, b any) Response {
	return Response{Status: s, Headers: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, Body: b}
}

// NoContent builds a 204 No Content response.
func NoContent() Response {
	return Response{Status: http.StatusNoContent}
}

// BindJSON decodes a JSON payload into dst, rejecting unknown fields.
func BindJSON(r io.Reader, dst any) error {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	return d.Decode(dst)
}
