package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

// Handler adapts an api.HandlerFunc into an api.Handler.
type Handler[T any] struct {
	HandleFunc func(context.Context, api.Request) (api.Response, error)
}

// Handle implements api.Handler.
func (h Handler[T]) Handle(ctx context.Context, request api.Request) (api.Response, error) {
	return h.HandleFunc(ctx, request)
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
