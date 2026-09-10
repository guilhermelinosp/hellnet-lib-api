package adapter

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	api "github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

// Router is a gin-backed implementation of api.Router.
type Router struct {
	engine          *gin.Engine
	root            *gin.RouterGroup
	logger          *slog.Logger
	bodyLimit       int64
	groupMiddleware []api.Middleware
}

var _ api.Router = (*Router)(nil)

// New builds a gin-backed Router from cfg.
func New(cfg Config) *Router {
	if cfg.ReleaseMode || cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	logger := cfg.Logger
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
	engine.NoRoute(func(c *gin.Context) {
		writeError(c, logger, apierrors.New(http.StatusNotFound, "NOT_FOUND", "route not found"))
	})
	engine.NoMethod(func(c *gin.Context) {
		writeError(c, logger, apierrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed for this resource"))
	})
	return &Router{engine: engine, root: &engine.RouterGroup, logger: logger, bodyLimit: limit, groupMiddleware: append([]api.Middleware(nil), cfg.GlobalMiddleware...)}
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
