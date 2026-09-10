package api

import (
	"context"
	"net/http"
	"runtime"

	"github.com/guilhermelinosp/hellnet-lib-api/config"
)

type PlatformHandlers struct { Live http.Handler; Ready http.Handler; Health http.Handler }
func (p PlatformHandlers) Valid() bool { return p.Live != nil && p.Ready != nil && p.Health != nil }
type ServiceInfo struct { Name string; Version string; Commit string; BuiltAt string }
type Deps struct { Platform PlatformHandlers; Routes []Route }
const ( PathRoot = "/"; PathLive = "/live"; PathReady = "/ready"; PathHealth = "/health"; Prefix = "/api/v1" )
func RegisterPlatformFromConfig(router Router, cfg *config.Config, deps Deps) { RegisterPlatform(router, ServiceInfo{Name: cfg.Name, Version: cfg.Build.Version, Commit: cfg.Build.Commit, BuiltAt: cfg.Build.Date}, deps) }
func RegisterPlatform(router Router, info ServiceInfo, deps Deps) { if deps.Platform.Live != nil { router.Mount(http.MethodGet, PathLive, deps.Platform.Live) }; if deps.Platform.Ready != nil { router.Mount(http.MethodGet, PathReady, deps.Platform.Ready) }; if deps.Platform.Health != nil { router.Mount(http.MethodGet, PathHealth, deps.Platform.Health) }; router.Handle(http.MethodGet, PathRoot, serviceInfoHandler(info)); v1:=router.Group(Prefix); for _, r:=range deps.Routes { switch { case r.Handler != nil: v1.Handle(r.Method,r.Path,r.Handler); case r.Raw != nil: v1.Mount(r.Method,r.Path,r.Raw) } } }
func serviceInfoHandler(info ServiceInfo) Handler { return HandlerFunc(func(_ context.Context,_ Request)(Response,error){ return JSON(http.StatusOK,map[string]string{"service":info.Name,"version":info.Version,"commit":info.Commit,"builtAt":info.BuiltAt,"go":runtime.Version(),"docs":"openapi/openapi.yaml"}),nil }) }
