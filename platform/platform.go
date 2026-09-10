// Package platform is the single entry point for a hellnet HTTP service.
//
// It wires the environment-driven config, the gin adapter (with telemetry
// instrumentation), the platform routes and the HTTP server in one place, so
// applications start with a single constructor instead of wiring
// config.New + adapter.New + telemetry.Middleware + server.New themselves.
package platform

import (
	"context"
	"errors"
	"log/slog"

	"github.com/guilhermelinosp/hellnet-lib-api/adapter"
	"github.com/guilhermelinosp/hellnet-lib-api/api"
	"github.com/guilhermelinosp/hellnet-lib-api/config"
	"github.com/guilhermelinosp/hellnet-lib-api/server"
	"github.com/guilhermelinosp/hellnet-lib-telemetry/telemetry"
)

// App is a fully wired HTTP application ready to register routes and run.
type App struct {
	Config *config.Config
	Logger *slog.Logger
	Router *adapter.Router
	Server *server.Server

	tel *telemetry.Telemetry
}

// New builds an App from a telemetry instance. It loads the environment-driven
// config, builds the gin adapter, instruments the handler tree with the
// telemetry middleware and mounts the HTTP server with graceful shutdown.
// A nil telemetry is not supported: callers must create it first.
func New(tel *telemetry.Telemetry) (*App, error) {
	if tel == nil {
		return nil, errors.New("platform: telemetry is required")
	}
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}
	logger := tel.Logger
	if logger == nil {
		logger = slog.Default()
	}
	router := adapter.New(cfg, logger)
	httpHandler := telemetry.Middleware(tel, router)
	srv := server.New(cfg, logger, httpHandler)
	return &App{Config: cfg, Logger: logger, Router: router, Server: srv, tel: tel}, nil
}

// PlatformHandlers returns the liveness, readiness and health handlers backed
// by the telemetry instance.
func (a *App) PlatformHandlers() api.PlatformHandlers {
	return api.PlatformHandlers{
		Live:   a.tel.Live(),
		Ready:  a.tel.Ready(),
		Health: a.tel.Health(),
	}
}

// Register mounts the platform routes and the given dependency routes.
func (a *App) Register(deps api.Deps) {
	api.RegisterPlatform(a.Router, deps)
}

// RegisterRoutes mounts only business routes under the versioned API prefix.
func (a *App) RegisterRoutes(routes []api.Route) {
	a.Register(api.Deps{Routes: routes})
}

// Run serves HTTP until ctx is cancelled, draining connections gracefully.
func (a *App) Run(ctx context.Context) error {
	return a.Server.Run(ctx)
}

// Shutdown flushes telemetry providers after the HTTP server has drained.
func (a *App) Shutdown() error {
	return a.tel.Shutdown()
}
