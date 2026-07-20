package handler

import "github.com/bendy/file-gateway/internal/types"

// AdminGitHubLogin handles POST /admin/api/v1/auth/github
func AdminGitHubLogin(req *types.Request) types.Response {
	return types.Error(501, "not_implemented", "admin login not yet implemented", nil)
}

// AdminMe handles GET /admin/api/v1/auth/me
func AdminMe(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "admin profile not yet implemented", nil)
}

// AdminLogout handles POST /admin/api/v1/auth/logout
func AdminLogout(req *types.Request) types.Response {
	return types.Error(501, "not_implemented", "admin logout not yet implemented", nil)
}

// AdminStats handles GET /admin/api/v1/stats
func AdminStats(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "admin stats not yet implemented", nil)
}

// AdminListTenants handles GET /admin/api/v1/tenants
func AdminListTenants(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant list not yet implemented", nil)
}

// AdminCreateTenant handles POST /admin/api/v1/tenants
func AdminCreateTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant creation not yet implemented", nil)
}

// AdminGetTenant handles GET /admin/api/v1/tenants/detail
func AdminGetTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant detail not yet implemented", nil)
}

// AdminUpdateTenant handles PATCH /admin/api/v1/tenants/update
func AdminUpdateTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant update not yet implemented", nil)
}

// AdminDeleteTenant handles DELETE /admin/api/v1/tenants/delete
func AdminDeleteTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant deletion not yet implemented", nil)
}

// AdminRotateKey handles POST /admin/api/v1/tenants/rotate-key
func AdminRotateKey(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "key rotation not yet implemented", nil)
}

// AdminGetTenantQuota handles GET /admin/api/v1/tenants/quota
func AdminGetTenantQuota(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "tenant quota not yet implemented", nil)
}

// AdminUpdateTenantQuota handles PATCH /admin/api/v1/tenants/quota
func AdminUpdateTenantQuota(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "quota update not yet implemented", nil)
}

// AdminListBackends handles GET /admin/api/v1/tenants/backends
func AdminListBackends(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "backend list not yet implemented", nil)
}

// AdminCreateBackend handles POST /admin/api/v1/tenants/backends
func AdminCreateBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "backend creation not yet implemented", nil)
}

// AdminUpdateBackend handles PATCH /admin/api/v1/backends/update
func AdminUpdateBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "backend update not yet implemented", nil)
}

// AdminDeleteBackend handles DELETE /admin/api/v1/backends/delete
func AdminDeleteBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}
	return types.Error(501, "not_implemented", "backend deletion not yet implemented", nil)
}
