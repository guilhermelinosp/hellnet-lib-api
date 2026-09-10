package api

import (
	"net/http"
)

// PlatformHandlers bundles the liveness, readiness and health handlers.
type PlatformHandlers struct {
	Live   http.Handler
	Ready  http.Handler
	Health http.Handler
}

// Valid reports whether all platform handlers are configured.
func (p PlatformHandlers) Valid() bool {
	return p.Live != nil && p.Ready != nil && p.Health != nil
}

// Deps carries the dependency wiring for RegisterPlatform.
type Deps struct {
	Platform PlatformHandlers
	Routes   []Route
}

const (
	// PathLive is the liveness probe endpoint.
	PathLive = "/live"
	// PathReady is the readiness probe endpoint.
	PathReady = "/ready"
	// PathHealth is the health endpoint.
	PathHealth = "/health"
	// Prefix is the API version prefix.
	Prefix = "/api/v1"
)

// RegisterPlatform mounts platform routes and the versioned API group.
func RegisterPlatform(router Router, deps Deps) {
	if deps.Platform.Live != nil {
		router.Mount(http.MethodGet, PathLive, deps.Platform.Live)
	}
	if deps.Platform.Ready != nil {
		router.Mount(http.MethodGet, PathReady, deps.Platform.Ready)
	}
	if deps.Platform.Health != nil {
		router.Mount(http.MethodGet, PathHealth, deps.Platform.Health)
	}
	v1 := router.Group(Prefix)
	for _, r := range deps.Routes {
		switch {
		case r.Handler != nil:
			v1.Handle(r.Method, r.Path, r.Handler)
		case r.Raw != nil:
			v1.Mount(r.Method, r.Path, r.Raw)
		}
	}
}
