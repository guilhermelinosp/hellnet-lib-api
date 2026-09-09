package adapter

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const (
	defaultBodyLimit int64 = 1 << 20
	envReleaseMode = "HELLNET_API_RELEASE_MODE"
	envBodyLimit = "HELLNET_API_BODY_LIMIT"
	envCORSOrigins = "HELLNET_API_CORS_ALLOWED_ORIGINS"
)

type Config struct {
	Logger *slog.Logger
	ReleaseMode bool
	CORSAllowedOrigins []string
	BodyLimit int64
	GlobalMiddleware []Middleware
}

func ConfigFromEnv(logger *slog.Logger) Config {
	cfg := Config{Logger: logger, BodyLimit: defaultBodyLimit, CORSAllowedOrigins: splitEnv(os.Getenv(envCORSOrigins))}
	if raw := strings.TrimSpace(os.Getenv(envReleaseMode)); raw != "" { cfg.ReleaseMode, _ = strconv.ParseBool(raw) }
	if raw := strings.TrimSpace(os.Getenv(envBodyLimit)); raw != "" { if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 { cfg.BodyLimit = n } }
	return cfg
}

func splitEnv(raw string) []string { var values []string; for _, item := range strings.Split(raw, ",") { if item = strings.TrimSpace(item); item != "" { values = append(values, item) } }; return values }
