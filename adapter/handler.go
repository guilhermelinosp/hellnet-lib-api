package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

type Request struct{ ctx *gin.Context }
func (r *Request) Param(name string) string { return r.ctx.Param(name) }
func (r *Request) Query(name string) string { return r.ctx.Query(name) }
func (r *Request) Header(name string) string { return r.ctx.GetHeader(name) }
func (r *Request) Raw() *http.Request { return r.ctx.Request }
func (r *Request) Bind(v any) error { err := api.BindJSON(r.ctx.Request.Body, v); if isTooLarge(err) { return apierrors.New(http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body exceeds allowed size") }; return err }
func writeResponse(c *gin.Context, resp api.Response) { status := resp.Status; if status == 0 { status = http.StatusOK }; for key, values := range resp.Headers { for _, value := range values { c.Writer.Header().Add(key, value) } }; if resp.Body == nil { c.Status(status); return }; payload, err := json.Marshal(resp.Body); if err != nil { writeError(c, slog.Default(), apierrors.Wrap(apierrors.Internal(), err)); return }; c.Data(status, "application/json; charset=utf-8", payload) }
type errorEnvelope struct { Error errorDetail `json:"error"`; RequestID string `json:"requestId,omitempty"` }
type errorDetail struct { Code string `json:"code"`; Message string `json:"message"` }
func writeError(c *gin.Context, logger *slog.Logger, err error) { mapped := apierrors.Map(err); if apierrors.Cause(mapped) != nil && !errors.Is(apierrors.Cause(mapped), context.Canceled) { logger.ErrorContext(c.Request.Context(), "request failed", slog.Any("error", apierrors.Cause(mapped))) }; c.Header("Content-Type", "application/json; charset=utf-8"); _ = c.Error(nil); payload, _ := json.Marshal(errorEnvelope{Error: errorDetail{Code: mapped.Code, Message: mapped.Message}, RequestID: requestIDFrom(c)}); c.Data(mapped.Status, "application/json; charset=utf-8", payload) }
func isTooLarge(err error) bool { var mbe *http.MaxBytesError; return errors.As(err, &mbe) }
