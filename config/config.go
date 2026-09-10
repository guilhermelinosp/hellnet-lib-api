// Package config provides environment-driven runtime configuration for
// Hellnet Go services, built on top of hellnet-lib-environments.
package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/guilhermelinosp/hellnet-lib-environments/environments"
)

// EnvPrefix is the primary environment variable prefix used by Config.
const EnvPrefix = "HELLNET_"

// envFallbackPrefix is the legacy prefix still honoured for compatibility.
const envFallbackPrefix = "APP_"

// Build holds build-time metadata injected at compile time.
type Build struct {
	Version string
	Commit  string
	Date    string
}

// Config holds runtime settings derived from environment variables.
type Config struct {
	Name               string
	Env                string
	Port               string
	ShutdownTimeout    time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ReadHeaderTimeout  time.Duration
	CORSAllowedOrigins []string
	BodyLimit          int64
	ReleaseMode        bool
	LogLevel           slog.Level
	LogFormat          string
	TrustedProxies     []string
	Build              Build
}

// FromEnv builds a Config from environment variables, applying defaults and
// validation. It loads a local .env in development environments (a no-op in
// production) and reads all values through hellnet-lib-environments, using
// EnvPrefix as the primary prefix and the legacy "APP_" prefix as fallback.
func FromEnv(build Build) (*Config, error) {
	if err := environments.LoadDotEnv(); err != nil {
		return nil, err
	}
	c := &Config{
		Name:               strings.TrimSpace(environments.GetString(EnvPrefix, envFallbackPrefix, "SERVICE", "")),
		Env:                strings.TrimSpace(environments.GetString(EnvPrefix, envFallbackPrefix, "ENV", "Development")),
		Port:               environments.GetString(EnvPrefix, envFallbackPrefix, "PORT", "8080"),
		ShutdownTimeout:    10 * time.Second,
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
		ReadHeaderTimeout:  10 * time.Second,
		CORSAllowedOrigins: list(environments.GetString(EnvPrefix, envFallbackPrefix, "CORS_ALLOWED_ORIGINS", "")),
		BodyLimit:          int64(1 << 20),
		ReleaseMode:        environments.GetBool(EnvPrefix, envFallbackPrefix, "RELEASE_MODE", false),
		LogLevel:           parseLevel(environments.GetString(EnvPrefix, envFallbackPrefix, "LOG_LEVEL", "")),
		LogFormat:          parseFormat(environments.GetString(EnvPrefix, envFallbackPrefix, "LOG_FORMAT", "text")),
		TrustedProxies:     list(environments.GetString(EnvPrefix, envFallbackPrefix, "TRUSTED_PROXIES", "")),
		Build:              build,
	}
	for _, timeout := range []struct {
		suffix string
		target *time.Duration
	}{
		{"SHUTDOWN_TIMEOUT", &c.ShutdownTimeout},
		{"READ_TIMEOUT", &c.ReadTimeout},
		{"WRITE_TIMEOUT", &c.WriteTimeout},
		{"IDLE_TIMEOUT", &c.IdleTimeout},
		{"READ_HEADER_TIMEOUT", &c.ReadHeaderTimeout},
	} {
		dur, err := environments.GetDurationE(EnvPrefix, envFallbackPrefix, timeout.suffix, *timeout.target)
		if err != nil {
			return nil, err
		}
		*timeout.target = dur
	}
	if n, err := environments.GetIntE(EnvPrefix, envFallbackPrefix, "BODY_LIMIT", int(1<<20)); err == nil && n > 0 {
		c.BodyLimit = int64(n)
	} else if err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate checks the configuration for invalid or inconsistent values.
func (c *Config) Validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("config: %sPORT %q is invalid", EnvPrefix, c.Port)
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("config: %sNAME cannot be empty", EnvPrefix)
	}
	if c.ShutdownTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 || c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("config: timeouts must be positive")
	}
	if c.BodyLimit <= 0 {
		return fmt.Errorf("config: %sBODY_LIMIT must be positive", EnvPrefix)
	}
	return nil
}

// IsProduction reports whether the environment is set to production.
func (c *Config) IsProduction() bool { return strings.EqualFold(c.Env, "Production") }

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json":
		return "json"
	default:
		return "text"
	}
}

func list(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
