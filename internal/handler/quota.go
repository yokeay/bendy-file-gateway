package handler

import (
	"github.com/bendy/file-gateway/internal/quota"
	"github.com/bendy/file-gateway/internal/types"
)

// GetQuota handles GET /api/v1/quota
func GetQuota(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	q, err := quota.GetQuota(req.TenantID)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.JSON(200, q)
}
