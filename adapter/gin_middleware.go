package adapter

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// requestID generates or reuses a request ID, propagating it in the response header.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := SanitizeRequestID(c.GetHeader(RequestIDHeader))
		if id == "" {
			id = GenerateRequestID()
		}
		c.Header(RequestIDHeader, id)
		c.Set("requestID", id)
		c.Next()
	}
}

// requestIDFrom retrieves the request ID previously set by the requestID middleware.
func requestIDFrom(c *gin.Context) string {
	if id, ok := c.Get("requestID"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// securityHeaders delegates to the hardening middleware from the adapter package.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		SecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { c.Next() })).ServeHTTP(c.Writer, c.Request)
	}
}

// cors delegates to the CORS middleware from the adapter package.
func cors(origins []string) gin.HandlerFunc {
	apply := CORS(origins)
	return func(c *gin.Context) {
		apply(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { c.Next() })).ServeHTTP(c.Writer, c.Request)
	}
}

// recovery logs panics with a sanitized path and aborts with 500.
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
