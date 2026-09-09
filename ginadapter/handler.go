package ginadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	api "github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
	"github.com/gin-gonic/gin"
)

type Request struct{ ctx *gin.Context }
func (r *Request) Param(name string) string { return r.ctx.Param(name) }
func (r *Request) Query(name string) string { return r.ctx.Query(name) }
func (r *Request) Header(name string) string { return r.ctx.GetHeader(name) }
func (r *Request) Raw() *http.Request { return r.ctx.Request }
func (r *Request) Bind(v any) error { err := api.BindJSON(r.ctx.Request.Body, v); if isTooLarge(err) { return apierrors.New(http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body exceeds allowed size") }; return err }

func writeResponse(c *gin.Context, resp api.Response) {
	status := resp.Status; if status == 0 { status = http.StatusOK }
	for key, values := range resp.Headers { for _, value := range values { c.Writer.Header().Add(key, value) } }
	if resp.Body == nil { c.Status(status); return }
	payload, err := json.Marshal(resp.Body); if err != nil { writeError(c, slog.Default(), apierrors.Wrap(apierrors.Internal(), err)); return }
	c.Data(status, "application/json; charset=utf-8", payload)
}

type errorEnvelope struct { Error errorDetail `json:"error"`; RequestID string `json:"requestId,omitempty"` }
type errorDetail struct { Code string `json:"code"`; Message string `json:"message"` }

func writeError(c *gin.Context, logger *slog.Logger, err error) {
	mapped := apierrors.Map(err)
	if apierrors.Cause(mapped) == nil && mapped.Status < http.StatusInternalServerError { logger.WarnContext(c.Request.Context(), "request rejected", slog.String("method", c.Request.Method), slog.String("path", c.Request.URL.Path), slog.Int("status", mapped.Status), slog.String("code", mapped.Code), slog.String("message", mapped.Message)) }
	if cause := apierrors.Cause(mapped); cause != nil && !errors.Is(cause, context.Canceled) { logger.ErrorContext(c.Request.Context(), "request failed", slog.String("method", c.Request.Method), slog.String("path", c.Request.URL.Path), slog.Int("status", mapped.Status), slog.String("code", mapped.Code), slog.Any("error", cause)) }
	payload, marshalErr := json.Marshal(errorEnvelope{Error: errorDetail{Code: mapped.Code, Message: mapped.Message}, RequestID: requestIDFrom(c)})
	if marshalErr != nil { c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", []byte(`{"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`)); return }
	c.Data(mapped.Status, "application/json; charset=utf-8", payload)
}

func isTooLarge(err error) bool { var mbe *http.MaxBytesError; return errors.As(err, &mbe) }
