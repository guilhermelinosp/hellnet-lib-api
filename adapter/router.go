package adapter

import (
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	api "github.com/guilhermelinosp/hellnet-lib-api/api"
	"github.com/guilhermelinosp/hellnet-lib-api/config"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

var wildcardPattern = regexp.MustCompile(`\{([a-zA-Z0-9_]+)}`)

// RouteRegistrar is the minimal route-mounting surface shared by adapters.
type RouteRegistrar interface {
	Handle(string, string, api.Handler, ...api.Middleware)
	Mount(string, string, http.Handler)
	Group(string, ...api.Middleware) api.Router
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// TranslatePath converts {param} placeholders to the adapter's path syntax.
func TranslatePath(path string) string {
	if !strings.Contains(path, "{") {
		return path
	}
	return wildcardPattern.ReplaceAllString(path, ":$1")
}

// Router is a gin-backed implementation of api.Router.
type Router struct {
	engine          *gin.Engine
	root            *gin.RouterGroup
	logger          *slog.Logger
	bodyLimit       int64
	groupMiddleware []api.Middleware
}

var _ api.Router = (*Router)(nil)

// New builds a gin-backed Router from the application config and an
// optional logger. The logger is created by the caller (typically
// hellnet-lib-telemetry) and is never nil at runtime.
func New(cfg *config.Config, logger *slog.Logger) *Router {
	if cfg.ReleaseMode || cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	if logger == nil {
		logger = slog.Default()
	}
	limit := cfg.BodyLimit
	if limit <= 0 {
		limit = 1 << 20
	}
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(requestID())
	engine.Use(securityHeaders())
	if len(cfg.CORSAllowedOrigins) > 0 {
		engine.Use(cors(cfg.CORSAllowedOrigins))
	}
	engine.Use(recovery(logger))
	applyTrustedProxies(engine, cfg.TrustedProxies)
	engine.NoRoute(func(c *gin.Context) {
		writeError(c, logger, apierrors.New(http.StatusNotFound, "NOT_FOUND", "route not found"))
	})
	engine.NoMethod(func(c *gin.Context) {
		writeError(c, logger, apierrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed for this resource"))
	})
	return &Router{engine: engine, root: &engine.RouterGroup, logger: logger, bodyLimit: limit}
}

// Handle registers an api.Handler at the given method and path.
func (r *Router) Handle(method, path string, handler api.Handler, middlewares ...api.Middleware) {
	r.root.Handle(method, TranslatePath(path), r.wrap(handler, middlewares))
}

// Mount registers a raw http.Handler at the given method and path.
func (r *Router) Mount(method, path string, rawHandler http.Handler) {
	r.root.Handle(method, TranslatePath(path), func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, r.bodyLimit)
		rawHandler.ServeHTTP(c.Writer, c.Request)
	})
}

// Group returns a sub-router with the given prefix and middlewares.
func (r *Router) Group(prefix string, middlewares ...api.Middleware) api.Router {
	group := &Router{engine: r.engine, root: r.root.Group(TranslatePath(prefix)), logger: r.logger, bodyLimit: r.bodyLimit, groupMiddleware: append(append([]api.Middleware(nil), r.groupMiddleware...), middlewares...)}
	return group
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, r.bodyLimit)
	r.engine.ServeHTTP(w, req)
}

// applyTrustedProxies restricts which proxies gin trusts when deriving the
// client IP from forwarded headers. An empty list means no proxy is trusted:
// gin then derives the client IP from the connection address, avoiding
// X-Forwarded-For spoofing on services that are not behind a proxy.
func applyTrustedProxies(engine *gin.Engine, trusted []string) {
	if len(trusted) == 0 {
		_ = engine.SetTrustedProxies(nil)
		return
	}
	_ = engine.SetTrustedProxies(trusted)
}

func (r *Router) wrap(handler api.Handler, routeMiddlewares []api.Middleware) gin.HandlerFunc {
	chain := append(append([]api.Middleware(nil), r.groupMiddleware...), routeMiddlewares...)
	wrapped := handler
	for _, middleware := range slices.Backward(chain) {
		wrapped = middleware(wrapped)
	}
	return func(c *gin.Context) {
		resp, err := wrapped.Handle(c.Request.Context(), &Request{ctx: c})
		if err != nil {
			writeError(c, r.logger, err)
			return
		}
		writeResponse(c, resp)
	}
}
