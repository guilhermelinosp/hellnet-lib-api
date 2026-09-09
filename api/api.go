package api

import("context";"encoding/json";"io";"net/http")
type Handler interface{Handle(context.Context,Request)(Response,error)}
type HandlerFunc func(context.Context,Request)(Response,error)
func(f HandlerFunc)Handle(c context.Context,r Request)(Response,error){return f(c,r)}
type Request interface{Param(string)string;Query(string)string;Header(string)string;Bind(any)error;Raw()*http.Request}
type Response struct{Status int;Headers http.Header;Body any}
type Route struct{Method string;Path string;Handler Handler;Raw http.Handler}
type Middleware func(Handler)Handler
type Router interface{Handle(string,string,Handler,...Middleware);Mount(string,string,http.Handler);Group(string,...Middleware)Router;ServeHTTP(http.ResponseWriter,*http.Request)}
func JSON(s int,b any)Response{return Response{Status:s,Headers:http.Header{"Content-Type":[]string{"application/json; charset=utf-8"}},Body:b}}
func NoContent()Response{return Response{Status:http.StatusNoContent}}
func BindJSON(r io.Reader,dst any)error{d:=json.NewDecoder(r);d.DisallowUnknownFields();return d.Decode(dst)}
