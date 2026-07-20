package handler

import "github.com/bendy/file-gateway/internal/server"

// CreateDirectory handles POST /api/v1/directories
func CreateDirectory(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "create directory not yet implemented", nil)
}

// ListDirectory handles GET /api/v1/directories
func ListDirectory(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "list directory not yet implemented", nil)
}

// DeleteDirectory handles DELETE /api/v1/directories
func DeleteDirectory(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "delete directory not yet implemented", nil)
}
