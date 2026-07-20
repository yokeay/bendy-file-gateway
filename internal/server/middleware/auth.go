package middleware

import (
	"strings"

	"github.com/bendy/file-gateway/internal/auth"
	"github.com/bendy/file-gateway/internal/server"
)

// Auth checks authentication for tenant API and admin API routes.
func Auth() server.Middleware {
	return func(next server.Handler) server.Handler {
		return func(req *server.Request) server.Response {
			path := strings.TrimRight(req.Path, "/")

			switch {
			case strings.HasPrefix(path, "/admin/"):
				return adminAuth(next, req)
			case strings.HasPrefix(path, "/api/"):
				return tenantAuth(next, req)
			default:
				// Public routes (health, etc.)
				return next(req)
			}
		}
	}
}

func tenantAuth(next server.Handler, req *server.Request) server.Response {
	result, err := auth.VerifyTenantRequest(req)
	if err != nil {
		return server.Error(401, "unauthorized", err.Error(), nil)
	}

	req.TenantID = result.TenantID
	req.AccessKey = result.AccessKey
	return next(req)
}

func adminAuth(next server.Handler, req *server.Request) server.Response {
	// Skip auth for login endpoint
	if strings.HasSuffix(req.Path, "/auth/github") {
		return next(req)
	}

	// Check session cookie
	cookie := req.Headers["cookie"]
	if cookie == "" {
		return server.Error(401, "unauthorized", "missing session cookie", nil)
	}

	sessionToken := extractCookie(cookie, "session_token")
	if sessionToken == "" {
		return server.Error(401, "unauthorized", "missing session token", nil)
	}

	adminID, err := auth.ValidateAdminSession(sessionToken)
	if err != nil {
		return server.Error(401, "unauthorized", err.Error(), nil)
	}

	req.AdminID = adminID
	req.IsAdmin = true
	return next(req)
}

func extractCookie(cookieHeader, name string) string {
	for _, c := range strings.Split(cookieHeader, ";") {
		c = strings.TrimSpace(c)
		parts := strings.SplitN(c, "=", 2)
		if len(parts) == 2 && parts[0] == name {
			return parts[1]
		}
	}
	return ""
}
