package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	api "github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

// BindJSON decodes a JSON request body, rejecting nil bodies.
func BindJSON(body io.Reader, target any) error {
	if body == nil {
		return apierrors.Validation("body", "is required")
	}
	return api.BindJSON(body, target)
}

// LimitBody caps the request body size using an http.MaxBytesReader.
func LimitBody(w http.ResponseWriter, r *http.Request, limit int64) {
	if limit > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}
}

// JSONHandler adapts a typed handler that decodes the request body into TReq
// and serializes TResp as JSON. Implementing api.Handler, it lets application
// code write endpoint logic without touching Request/Response plumbing.
type JSONHandler[TReq any, TResp any] struct {
	HandleFunc func(context.Context, *TReq) (TResp, error)
}

// Handle implements api.Handler by binding the body to TReq, delegating to
// HandleFunc, and wrapping the result as a JSON response.
func (h JSONHandler[TReq, TResp]) Handle(ctx context.Context, request api.Request) (api.Response, error) {
	var req TReq
	if err := request.Bind(&req); err != nil {
		return api.Response{}, err
	}
	resp, err := h.HandleFunc(ctx, &req)
	if err != nil {
		return api.Response{}, err
	}
	return api.JSON(http.StatusOK, resp), nil
}

// Request adapts a gin.Context to the api.Request interface.
type Request struct{ ctx *gin.Context }

// Param returns the named path parameter.
func (r *Request) Param(name string) string { return r.ctx.Param(name) }

// Query returns the named query parameter.
func (r *Request) Query(name string) string { return r.ctx.Query(name) }

// Header returns the named request header.
func (r *Request) Header(name string) string { return r.ctx.GetHeader(name) }

// Raw returns the underlying *http.Request.
func (r *Request) Raw() *http.Request { return r.ctx.Request }

// Bind decodes the request body into v.
func (r *Request) Bind(v any) error {
	err := api.BindJSON(r.ctx.Request.Body, v)
	if isTooLarge(err) {
		return apierrors.New(http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body exceeds allowed size")
	}
	return err
}

// ErrorEnvelope is the JSON error envelope written to clients.
type ErrorEnvelope struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"requestId,omitempty"`
}

// ErrorDetail is the inner error payload of ErrorEnvelope.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteResponse serializes a Response to the standard http.ResponseWriter.
func WriteResponse(w http.ResponseWriter, response api.Response) {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	for key, values := range response.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if response.Body == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response.Body); err != nil {
		WriteError(w, slog.Default(), apierrors.Wrap(apierrors.Internal(), err))
	}
}

// WriteError maps err via apierrors and writes the error envelope.
func WriteError(w http.ResponseWriter, logger *slog.Logger, err error) {
	mapped := apierrors.Map(err)
	if mapped == nil {
		mapped = apierrors.Internal()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cause := apierrors.Cause(mapped); cause != nil && !errors.Is(cause, context.Canceled) {
		logger.Error("request failed", slog.Any("error", cause))
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(mapped.Status)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope{Error: ErrorDetail{Code: mapped.Code, Message: mapped.Message}})
}

func writeResponse(c *gin.Context, resp api.Response) {
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	for key, values := range resp.Headers {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	if resp.Body == nil {
		c.Status(status)
		return
	}
	payload, err := json.Marshal(resp.Body)
	if err != nil {
		writeError(c, slog.Default(), apierrors.Wrap(apierrors.Internal(), err))
		return
	}
	c.Data(status, "application/json; charset=utf-8", payload)
}

func writeError(c *gin.Context, logger *slog.Logger, err error) {
	mapped := apierrors.Map(err)
	path := sanitizeForLog(c.Request.URL.Path)
	if apierrors.Cause(mapped) == nil && mapped.Status < http.StatusInternalServerError {
		logger.WarnContext(c.Request.Context(), "request rejected", slog.String("method", c.Request.Method), slog.String("path", path), slog.Int("status", mapped.Status), slog.String("code", mapped.Code), slog.String("message", mapped.Message))
	}
	if cause := apierrors.Cause(mapped); cause != nil && !errors.Is(cause, context.Canceled) {
		logger.ErrorContext(c.Request.Context(), "request failed", slog.String("method", c.Request.Method), slog.String("path", path), slog.Int("status", mapped.Status), slog.String("code", mapped.Code), slog.Any("error", cause))
	}
	payload, marshalErr := json.Marshal(ErrorEnvelope{Error: ErrorDetail{Code: mapped.Code, Message: mapped.Message}, RequestID: requestIDFrom(c)})
	if marshalErr != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", []byte(`{"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
		return
	}
	c.Data(mapped.Status, "application/json; charset=utf-8", payload)
}

func sanitizeForLog(value string) string {
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	return value
}

func isTooLarge(err error) bool { var mbe *http.MaxBytesError; return errors.As(err, &mbe) }
