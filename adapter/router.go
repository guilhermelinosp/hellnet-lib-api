package adapter

import (
	"log/slog"
	"net/http"
	"os"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/guilhermelinosp/hellnet-lib-api/adapter"
	api "github.com/guilhermelinosp/hellnet-lib-api/api"
	apierrors "github.com/guilhermelinosp/hellnet-lib-api/errors"
)

type Config = adapter.Config
type Router struct { engine *gin.Engine; root *gin.RouterGroup; logger *slog.Logger; bodyLimit int64; groupMiddleware []api.Middleware }
var _ api.Router = (*Router)(nil)
func New(cfg Config) *Router { if cfg.ReleaseMode { gin.SetMode(gin.ReleaseMode) } else if os.Getenv("GIN_MODE") == "" { gin.SetMode(gin.DebugMode) }; logger := cfg.Logger; if logger == nil { logger = slog.Default() }; limit := cfg.BodyLimit; if limit <= 0 { limit = 1 << 20 }; engine := gin.New(); engine.HandleMethodNotAllowed = true; engine.Use(requestID(), securityHeaders()); if len(cfg.CORSAllowedOrigins) > 0 { engine.Use(cors(cfg.CORSAllowedOrigins)) }; engine.Use(recovery(logger)); engine.NoRoute(func(c *gin.Context) { writeError(c, logger, apierrors.NotFound("route")) }); engine.NoMethod(func(c *gin.Context) { writeError(c, logger, apierrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed for this resource")) }); return &Router{engine: engine, root: &engine.RouterGroup, logger: logger, bodyLimit: limit, groupMiddleware: append([]api.Middleware(nil), cfg.GlobalMiddleware...)} }
func (r *Router) Handle(method, path string, handler api.Handler, middlewares ...api.Middleware) { r.root.Handle(method, adapter.TranslatePath(path), r.wrap(handler, middlewares)) }
func (r *Router) Mount(method, path string, rawHandler http.Handler) { r.root.Handle(method, adapter.TranslatePath(path), func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, r.bodyLimit); rawHandler.ServeHTTP(c.Writer, c.Request) }) }
func (r *Router) Group(prefix string, middlewares ...api.Middleware) api.Router { return &Router{engine:r.engine, root:r.root.Group(adapter.TranslatePath(prefix)), logger:r.logger, bodyLimit:r.bodyLimit, groupMiddleware:append(append([]api.Middleware(nil),r.groupMiddleware...),middlewares...)} }
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) { req.Body = http.MaxBytesReader(w, req.Body, r.bodyLimit); r.engine.ServeHTTP(w, req) }
func (r *Router) wrap(handler api.Handler, routeMiddlewares []api.Middleware) gin.HandlerFunc { chain:=append(append([]api.Middleware(nil),r.groupMiddleware...),routeMiddlewares...); wrapped:=handler; for _, middleware:=range slices.Backward(chain){wrapped=middleware(wrapped)}; return func(c *gin.Context){resp,err:=wrapped.Handle(c.Request.Context(),&Request{ctx:c}); if err!=nil{writeError(c,r.logger,err);return}; writeResponse(c,resp)} }
