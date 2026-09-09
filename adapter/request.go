package adapter

import (
	"io"
	"net/http"

	"github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

func BindJSON(body io.Reader, target any) error {
	if body == nil { return apierrors.Validation("body", "is required") }
	return api.BindJSON(body, target)
}

func LimitBody(w http.ResponseWriter, r *http.Request, limit int64) { if limit > 0 { r.Body = http.MaxBytesReader(w, r.Body, limit) } }
