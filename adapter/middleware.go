package adapter

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

// RequestIDHeader is the canonical request ID header name.
const RequestIDHeader = "X-Request-ID"

// SecurityHeaders writes hardening response headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// RequestID propagates a sanitized or generated request ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := SanitizeRequestID(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = GenerateRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

// SanitizeRequestID validates and normalizes an incoming request ID header value.
func SanitizeRequestID(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > 64 {
		return ""
	}
	for _, r := range raw {
		if !validIDChar(r) {
			return ""
		}
	}
	return raw
}

// validIDChar reports whether r is allowed in a request ID: alphanumeric,
// hyphen, underscore or dot.
func validIDChar(r rune) bool {
	return r >= '0' && r <= '9' ||
		r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r == '-' || r == '_' || r == '.'
}

// GenerateRequestID produces a random 128-bit hex request ID.
func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(b)
}

// CORS returns a middleware that allows the listed origins.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	wildcard := false
	for _, origin := range origins {
		if origin == "*" {
			wildcard = true
		}
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (wildcard || HasOrigin(allowed, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, "+RequestIDHeader)
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HasOrigin reports whether origin is explicitly allowed.
func HasOrigin(allowed map[string]struct{}, origin string) bool { _, ok := allowed[origin]; return ok }

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

// recovery logs panics with a sanitized path and writes a JSON error envelope
// so client-facing error handling stays consistent across the app.
func recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered", slog.String("method", c.Request.Method), slog.String("path", sanitizeForLog(c.Request.URL.Path)), slog.Any("panic", err))
				writeError(c, logger, apierrors.Internal())
				c.Abort()
			}
		}()
		c.Next()
	}
}
