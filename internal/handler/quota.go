package handler

import (
	"github.com/bendy/file-gateway/internal/quota"
	"github.com/bendy/file-gateway/internal/server"
)

// GetQuota handles GET /api/v1/quota
func GetQuota(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	q, err := quota.GetQuota(req.TenantID)
	if err != nil {
		return server.Error(500, "internal_error", err.Error(), nil)
	}

	return server.JSON(200, q)
}
