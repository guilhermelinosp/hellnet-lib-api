// Package config provides environment-driven runtime configuration for
// Hellnet Go services, built on top of hellnet-lib-environments.
//
// Following the same env-first pattern as hellnet-lib-telemetry and
// hellnet-lib-kafka: New() reads everything from the environment (.env in dev
// plus HELLNET_* with APP_* fallback), applies inline defaults, and never
// fails on missing values — validation happens afterwards.
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

// New builds runtime configuration from environment variables. It mirrors the
// telemetry/kafka constructor style: no parameters, env-first, best-effort
// .env loading, defaults inline, and no build metadata.
func New() (*Config, error) {
	return FromEnv(Build{})
}

// MustNew is like New but panics on error. Use at startup.
func MustNew() *Config {
	cfg, err := New()
	if err != nil {
		panic(err)
	}
	return cfg
}

// Values holds an explicit, non-environment configuration. Use WithValues to
// build a Config programmatically (tests, CLI tools, embedded servers) when
// environment variables are not the source of truth.
type Values struct {
	Name               string
	Env                string
	Port               string
	ShutdownTimeout    time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ReadHeaderTimeout  time.Duration
	CORSAllowedOrigins []string
	TrustedProxies     []string
	BodyLimit          int64
	LogFormat          string
	Build              Build
}

// WithValues builds a Config from explicit Values, applying the same inline
// defaults and derived fields (release mode and log level follow Env) as
// FromEnv, and validating the result exactly like the env-driven path.
func WithValues(v Values) (*Config, error) {
	env := strings.TrimSpace(v.Env)
	if env == "" {
		env = "Development"
	}
	c := &Config{
		Name:               strings.TrimSpace(v.Name),
		Env:                env,
		Port:               defaultString(v.Port, "8080"),
		ShutdownTimeout:    defaultDuration(v.ShutdownTimeout, 10*time.Second),
		ReadTimeout:        defaultDuration(v.ReadTimeout, 15*time.Second),
		WriteTimeout:       defaultDuration(v.WriteTimeout, 30*time.Second),
		IdleTimeout:        defaultDuration(v.IdleTimeout, 120*time.Second),
		ReadHeaderTimeout:  defaultDuration(v.ReadHeaderTimeout, 10*time.Second),
		CORSAllowedOrigins: v.CORSAllowedOrigins,
		BodyLimit:          defaultInt64(v.BodyLimit, 1<<20),
		ReleaseMode:        releaseModeFor(env),
		LogLevel:           logLevelFor(env),
		LogFormat:          parseFormat(v.LogFormat),
		TrustedProxies:     v.TrustedProxies,
		Build:              v.Build,
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultDuration(v, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	return v
}

func defaultInt64(v, fallback int64) int64 {
	if v <= 0 {
		return fallback
	}
	return v
}

// FromEnv builds a Config from environment variables, applying defaults and
// validation. It loads a local .env in development environments (best-effort,
// a no-op when missing or in production) and reads all values through
// hellnet-lib-environments, using EnvPrefix as the primary prefix and the
// legacy "APP_" prefix as fallback. Invalid individual values fall back to
// their inline default; only Validate can fail the whole configuration.
func FromEnv(build Build) (*Config, error) {
	_ = environments.LoadDotEnv() // best-effort, same as telemetry/kafka

	env := strings.TrimSpace(environments.GetString(EnvPrefix, envFallbackPrefix, "ENVIRONMENT", "Development"))
	c := &Config{
		Name:               strings.TrimSpace(environments.GetString(EnvPrefix, envFallbackPrefix, "SERVICE", "")),
		Env:                env,
		Port:               environments.GetString(EnvPrefix, envFallbackPrefix, "PORT", "8080"),
		ShutdownTimeout:    environments.GetDuration(EnvPrefix, envFallbackPrefix, "SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:        environments.GetDuration(EnvPrefix, envFallbackPrefix, "READ_TIMEOUT", 15*time.Second),
		WriteTimeout:       environments.GetDuration(EnvPrefix, envFallbackPrefix, "WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:        environments.GetDuration(EnvPrefix, envFallbackPrefix, "IDLE_TIMEOUT", 120*time.Second),
		ReadHeaderTimeout:  environments.GetDuration(EnvPrefix, envFallbackPrefix, "READ_HEADER_TIMEOUT", 10*time.Second),
		CORSAllowedOrigins: list(environments.GetString(EnvPrefix, envFallbackPrefix, "CORS_ALLOWED_ORIGINS", "")),
		BodyLimit:          int64(environments.GetInt(EnvPrefix, envFallbackPrefix, "BODY_LIMIT", 1<<20)),
		ReleaseMode:        releaseModeFor(env),
		LogLevel:           logLevelFor(env),
		LogFormat:          parseFormat(environments.GetString(EnvPrefix, envFallbackPrefix, "LOG_FORMAT", "text")),
		TrustedProxies:     list(environments.GetString(EnvPrefix, envFallbackPrefix, "TRUSTED_PROXIES", "")),
		Build:              build,
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
		return fmt.Errorf("config: %sSERVICE cannot be empty", EnvPrefix)
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
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "Production")
}

// logLevelFor derives the slog level from the environment: Debug in
// development, Info everywhere else. The level is never read from an
// environment variable — it follows HELLNET_ENVIRONMENT by design.
func logLevelFor(env string) slog.Level {
	if strings.EqualFold(strings.TrimSpace(env), "Development") {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// releaseModeFor derives the Gin release mode from the environment: enabled
// for anything that is not Development. Never read from an environment
// variable — it follows HELLNET_ENVIRONMENT by design.
func releaseModeFor(env string) bool {
	return !strings.EqualFold(strings.TrimSpace(env), "Development")
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
