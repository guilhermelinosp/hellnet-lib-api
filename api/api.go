// Package api contains framework-neutral contracts for embeddable Go APIs.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type Handler interface { Handle(context.Context, Request) (Response, error) }
type HandlerFunc func(context.Context, Request) (Response, error)
func (f HandlerFunc) Handle(ctx context.Context, req Request) (Response, error) { return f(ctx, req) }

type Request interface {
	Param(string) string
	Query(string) string
	Header(string) string
	Bind(any) error
	Raw() *http.Request
}

type Response struct { Status int; Headers http.Header; Body any }
type Route struct { Method string; Path string; Handler Handler; Raw http.Handler }
type Middleware func(Handler) Handler

type Router interface {
	Handle(string, string, Handler, ...Middleware)
	Mount(string, string, http.Handler)
	Group(string, ...Middleware) Router
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func JSON(status int, body any) Response {
	return Response{Status: status, Headers: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, Body: body}
}
func NoContent() Response { return Response{Status: http.StatusNoContent} }
func BindJSON(r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
