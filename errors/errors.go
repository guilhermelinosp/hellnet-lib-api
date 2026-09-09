package errors
import("errors";"fmt";"net/http")
type Error struct{Status int;Code string;Message string;cause error}
func(e *Error)Error()string{return e.Code+": "+e.Message}
func(e *Error)Unwrap()error{return e.cause}
func New(s int,c,m string)*Error{return &Error{Status:s,Code:c,Message:m}}
func Wrap(e *Error,c error)*Error{return &Error{Status:e.Status,Code:e.Code,Message:e.Message,cause:c}}
func Cause(e *Error)error{if e==nil{return nil};return e.cause}
func Map(e error)*Error{if e==nil{return nil};var a *Error;if errors.As(e,&a){return a};return Wrap(Internal(),e)}
func Validation(f,r string)*Error{return New(http.StatusBadRequest,"VALIDATION_ERROR",fmt.Sprintf("field %q %s",f,r))}
func Internal()*Error{return New(http.StatusInternalServerError,"INTERNAL_ERROR","internal server error")}
