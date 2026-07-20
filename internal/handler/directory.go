package handler

import "github.com/bendy/file-gateway/internal/types"

// CreateDirectory handles POST /api/v1/directories
func CreateDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return types.Error(501, "not_implemented", "create directory not yet implemented", nil)
}

// ListDirectory handles GET /api/v1/directories
func ListDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return types.Error(501, "not_implemented", "list directory not yet implemented", nil)
}

// DeleteDirectory handles DELETE /api/v1/directories
func DeleteDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return types.Error(501, "not_implemented", "delete directory not yet implemented", nil)
}
