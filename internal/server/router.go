package server

import (
	"strings"

	"github.com/bendy/file-gateway/internal/types"
)

// Route represents a registered route.
type Route struct {
	Method  string
	Pattern string
	Handler types.Handler
}

var routes []Route

// RegisterRoute adds a route to the route table.
func RegisterRoute(method, pattern string, handler types.Handler) {
	routes = append(routes, Route{method, pattern, handler})
}

// router returns the main request handler that dispatches to registered routes.
func router() types.Handler {
	return func(req *types.Request) types.Response {
		path := strings.TrimRight(req.Path, "/")
		if path == "" {
			path = "/"
		}

		for _, route := range routes {
			if route.Method != req.Method {
				continue
			}
			if route.Pattern == path {
				return route.Handler(req)
			}
		}

		// Try prefix match (for routes with dynamic segments handled in handler)
		for _, route := range routes {
			if route.Method != req.Method {
				continue
			}
			if strings.HasPrefix(path, route.Pattern) {
				return route.Handler(req)
			}
		}

		return types.Error(404, "not_found", "route not found: "+req.Method+" "+req.Path, nil)
	}
}
