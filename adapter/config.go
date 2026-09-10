package adapter

import (
	"log/slog"

	"github.com/guilhermelinosp/hellnet-lib-api/api"
	"github.com/guilhermelinosp/hellnet-lib-api/config"
)

// Config holds tunable runtime settings for an adapter. It embeds
// config.Config — the single source of environment-driven values — and adds
// the framework-specific wiring (logger and global middleware).
type Config struct {
	config.Config
	Logger           *slog.Logger
	GlobalMiddleware []api.Middleware
}

// ConfigFromEnv builds a Config from environment variables through
// config.FromEnv (hellnet-lib-environments with the HELLNET_ prefix).
func ConfigFromEnv(logger *slog.Logger) (*Config, error) {
	cfg, err := config.FromEnv(config.Build{})
	if err != nil {
		return nil, err
	}
	return &Config{Config: *cfg, Logger: logger}, nil
}

// FromConfig builds an adapter Config from an already-loaded config.Config.
func FromConfig(cfg config.Config, logger *slog.Logger) *Config {
	return &Config{Config: cfg, Logger: logger}
}
