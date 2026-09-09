package adapter

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/guilhermelinosp/hellnet-lib-api/api"
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
