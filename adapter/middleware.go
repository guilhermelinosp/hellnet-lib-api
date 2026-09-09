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

const requestIDHeader = "X-Request-ID"
const requestIDContextKey = "adapter.requestID"
func requestID() gin.HandlerFunc { return func(c *gin.Context) { id := strings.TrimSpace(c.GetHeader(requestIDHeader)); if id == "" { var b [16]byte; if _, err := rand.Read(b[:]); err == nil { id = hex.EncodeToString(b[:]) } else { id = "unavailable" } }; c.Set(requestIDContextKey, id); c.Writer.Header().Set(requestIDHeader, id); c.Next() } }
func requestIDFrom(c *gin.Context) string { value, ok := c.Get(requestIDContextKey); if !ok { return "" }; id, _ := value.(string); return id }
func securityHeaders() gin.HandlerFunc { return func(c *gin.Context) { for key, value := range map[string]string{"X-Content-Type-Options":"nosniff", "X-Frame-Options":"DENY", "Referrer-Policy":"strict-origin-when-cross-origin", "Content-Security-Policy":"default-src 'none'; frame-ancestors 'none'", "Cross-Origin-Resource-Policy":"same-origin"} { c.Writer.Header().Set(key, value) }; c.Next() } }
func cors(origins []string) gin.HandlerFunc { allowed := map[string]struct{}{}; wildcard := false; for _, origin := range origins { allowed[origin] = struct{}{}; wildcard = wildcard || origin == "*" }; return func(c *gin.Context) { origin := c.GetHeader("Origin"); if origin != "" && (wildcard || hasOrigin(allowed, origin)) { c.Header("Access-Control-Allow-Origin", origin); c.Header("Vary", "Origin"); c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS"); c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, "+requestIDHeader); c.Header("Access-Control-Max-Age", "600") }; if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" { c.AbortWithStatus(http.StatusNoContent); return }; c.Next() } }
func hasOrigin(allowed map[string]struct{}, origin string) bool { _, ok := allowed[origin]; return ok }
func recovery(logger *slog.Logger) gin.HandlerFunc { return func(c *gin.Context) { defer func() { if rec := recover(); rec != nil { logger.ErrorContext(c.Request.Context(), "panic recovered", slog.Any("panic", rec)); writeError(c, logger, apierrors.Internal()) } }(); c.Next() } }
