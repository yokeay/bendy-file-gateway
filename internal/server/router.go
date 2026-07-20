package server

import (
	"strings"

	"github.com/bendy/file-gateway/internal/handler"
)

// Route represents a registered route.
type Route struct {
	Method  string
	Pattern string
	Handler Handler
}

var routes []Route

func init() {
	routes = []Route{
		// Health check
		{"GET", "/health", handler.Health},

		// Tenant API - files
		{"POST", "/api/v1/files/upload", handler.UploadFile},
		{"GET", "/api/v1/files/download", handler.DownloadFile},
		{"GET", "/api/v1/files/info", handler.FileInfo},
		{"GET", "/api/v1/files/list", handler.ListFiles},
		{"DELETE", "/api/v1/files/delete", handler.DeleteFile},

		// Tenant API - directories
		{"POST", "/api/v1/directories", handler.CreateDirectory},
		{"GET", "/api/v1/directories", handler.ListDirectory},
		{"DELETE", "/api/v1/directories", handler.DeleteDirectory},

		// Tenant API - quota
		{"GET", "/api/v1/quota", handler.GetQuota},

		// Admin API - auth
		{"POST", "/admin/api/v1/auth/github", handler.AdminGitHubLogin},
		{"GET", "/admin/api/v1/auth/me", handler.AdminMe},
		{"POST", "/admin/api/v1/auth/logout", handler.AdminLogout},

		// Admin API - stats
		{"GET", "/admin/api/v1/stats", handler.AdminStats},

		// Admin API - tenants
		{"GET", "/admin/api/v1/tenants", handler.AdminListTenants},
		{"POST", "/admin/api/v1/tenants", handler.AdminCreateTenant},
		{"GET", "/admin/api/v1/tenants/detail", handler.AdminGetTenant},
		{"PATCH", "/admin/api/v1/tenants/update", handler.AdminUpdateTenant},
		{"DELETE", "/admin/api/v1/tenants/delete", handler.AdminDeleteTenant},
		{"POST", "/admin/api/v1/tenants/rotate-key", handler.AdminRotateKey},

		// Admin API - quota
		{"GET", "/admin/api/v1/tenants/quota", handler.AdminGetTenantQuota},
		{"PATCH", "/admin/api/v1/tenants/quota", handler.AdminUpdateTenantQuota},

		// Admin API - backends
		{"GET", "/admin/api/v1/tenants/backends", handler.AdminListBackends},
		{"POST", "/admin/api/v1/tenants/backends", handler.AdminCreateBackend},
		{"PATCH", "/admin/api/v1/backends/update", handler.AdminUpdateBackend},
		{"DELETE", "/admin/api/v1/backends/delete", handler.AdminDeleteBackend},
	}
}

// router returns the main request handler that dispatches to registered routes.
func router() Handler {
	return func(req *Request) Response {
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

		return Error(404, "not_found", "route not found: "+req.Method+" "+req.Path, nil)
	}
}
