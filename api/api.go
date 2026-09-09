// Package api contains small transport-neutral contracts for Go APIs.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Handler processes an API request.
type Handler interface {
	Handle(context.Context, Request) (Response, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, Request) (Response, error)

// Handle invokes f.
func (f HandlerFunc) Handle(ctx context.Context, req Request) (Response, error) { return f(ctx, req) }

// Request contains transport-neutral request accessors.
type Request interface {
	Param(string) string
	Query(string) string
	Header(string) string
	Bind(any) error
	Raw() *http.Request
}

// Response is a transport-neutral API response.
type Response struct {
	Status  int
	Headers http.Header
	Body    any
}

// Route associates a method and path with a handler.
type Route struct {
	Method  string
	Path    string
	Handler Handler
	Raw     http.Handler
}

// Middleware decorates a Handler.
type Middleware func(Handler) Handler

// Router registers and serves API routes.
type Router interface {
	Handle(string, string, Handler, ...Middleware)
	Mount(string, string, http.Handler)
	Group(string, ...Middleware) Router
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// JSON creates a JSON response.
func JSON(status int, body any) Response {
	return Response{Status: status, Headers: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, Body: body}
}

// NoContent creates an empty 204 response.
func NoContent() Response { return Response{Status: http.StatusNoContent} }

// BindJSON decodes JSON while rejecting unknown fields.
func BindJSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
