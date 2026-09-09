package ginadapter

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guilhermelinosp/hellnet-lib-api/adapter"
)

// requestID gera ou reutiliza um request ID, propagando-o no header de resposta.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := adapter.SanitizeRequestID(c.GetHeader(adapter.RequestIDHeader))
		if id == "" {
			id = adapter.GenerateRequestID()
		}
		c.Header(adapter.RequestIDHeader, id)
		c.Set("requestID", id)
		c.Next()
	}
}

// requestIDFrom recupera o request ID previamente definido pelo middleware requestID.
func requestIDFrom(c *gin.Context) string {
	if id, ok := c.Get("requestID"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// securityHeaders delega ao middleware hardening do pacote adapter.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		adapter.SecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { c.Next() })).ServeHTTP(c.Writer, c.Request)
	}
}

// cors delega ao middleware CORS do pacote adapter.
func cors(origins []string) gin.HandlerFunc {
	apply := adapter.CORS(origins)
	return func(c *gin.Context) {
		apply(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { c.Next() })).ServeHTTP(c.Writer, c.Request)
	}
}

func recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered", slog.String("method", c.Request.Method), slog.String("path", sanitizeForLog(c.Request.URL.Path)), slog.Any("panic", err))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
