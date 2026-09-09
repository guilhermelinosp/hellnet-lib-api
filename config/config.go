package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	Build              Build
}

// Default values used when the corresponding environment variables are unset.
const (
	DefaultName             = "service"
	DefaultEnv              = "Development"
	DefaultPort             = "8080"
	DefaultBodyLimit  int64 = 1 << 20
	DefaultShutdown         = 10 * time.Second
	DefaultReadHeader       = 10 * time.Second
	DefaultRead             = 15 * time.Second
	DefaultWrite            = 30 * time.Second
	DefaultIdle             = 120 * time.Second
)

// FromEnv builds a Config from environment variables, applying defaults and validation.
func FromEnv(build Build) (*Config, error) {
	c := &Config{
		Name:               value("APP_NAME", DefaultName),
		Env:                strings.TrimSpace(value("APP_ENV", DefaultEnv)),
		Port:               value("APP_PORT", DefaultPort),
		ShutdownTimeout:    DefaultShutdown,
		ReadTimeout:        DefaultRead,
		WriteTimeout:       DefaultWrite,
		IdleTimeout:        DefaultIdle,
		ReadHeaderTimeout:  DefaultReadHeader,
		CORSAllowedOrigins: list(value("APP_CORS_ALLOWED_ORIGINS", "")),
		BodyLimit:          DefaultBodyLimit,
		Build:              build,
	}
	var err error
	for key, target := range map[string]*time.Duration{
		"APP_SHUTDOWN_TIMEOUT":    &c.ShutdownTimeout,
		"APP_READ_TIMEOUT":        &c.ReadTimeout,
		"APP_WRITE_TIMEOUT":       &c.WriteTimeout,
		"APP_IDLE_TIMEOUT":        &c.IdleTimeout,
		"APP_READ_HEADER_TIMEOUT": &c.ReadHeaderTimeout,
	} {
		if *target, err = duration(key, *target); err != nil {
			return nil, err
		}
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
		return fmt.Errorf("config: APP_PORT %q is invalid", c.Port)
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("config: APP_NAME cannot be empty")
	}
	if c.ShutdownTimeout <= 0 || c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 || c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("config: timeouts must be positive")
	}
	return nil
}

// IsProduction reports whether the environment is set to production.
func (c *Config) IsProduction() bool { return strings.EqualFold(c.Env, "Production") }
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s=%q is invalid: %w", key, raw, err)
	}
	return d, nil
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
