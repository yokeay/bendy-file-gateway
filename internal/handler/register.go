package handler

import "github.com/bendy/file-gateway/internal/server"

func init() {
	// Health check
	server.RegisterRoute("GET", "/health", Health)

	// Tenant API - files
	server.RegisterRoute("POST", "/api/v1/files/upload", UploadFile)
	server.RegisterRoute("GET", "/api/v1/files/download", DownloadFile)
	server.RegisterRoute("GET", "/api/v1/files/info", FileInfo)
	server.RegisterRoute("GET", "/api/v1/files/list", ListFiles)
	server.RegisterRoute("DELETE", "/api/v1/files/delete", DeleteFile)

	// Tenant API - directories
	server.RegisterRoute("POST", "/api/v1/directories", CreateDirectory)
	server.RegisterRoute("GET", "/api/v1/directories", ListDirectory)
	server.RegisterRoute("DELETE", "/api/v1/directories", DeleteDirectory)

	// Tenant API - quota
	server.RegisterRoute("GET", "/api/v1/quota", GetQuota)

	// Admin API - auth
	server.RegisterRoute("POST", "/admin/api/v1/auth/github", AdminGitHubLogin)
	server.RegisterRoute("GET", "/admin/api/v1/auth/me", AdminMe)
	server.RegisterRoute("POST", "/admin/api/v1/auth/logout", AdminLogout)

	// Admin API - stats
	server.RegisterRoute("GET", "/admin/api/v1/stats", AdminStats)

	// Admin API - tenants
	server.RegisterRoute("GET", "/admin/api/v1/tenants", AdminListTenants)
	server.RegisterRoute("POST", "/admin/api/v1/tenants", AdminCreateTenant)
	server.RegisterRoute("GET", "/admin/api/v1/tenants/detail", AdminGetTenant)
	server.RegisterRoute("PATCH", "/admin/api/v1/tenants/update", AdminUpdateTenant)
	server.RegisterRoute("DELETE", "/admin/api/v1/tenants/delete", AdminDeleteTenant)
	server.RegisterRoute("POST", "/admin/api/v1/tenants/rotate-key", AdminRotateKey)

	// Admin API - quota
	server.RegisterRoute("GET", "/admin/api/v1/tenants/quota", AdminGetTenantQuota)
	server.RegisterRoute("PATCH", "/admin/api/v1/tenants/quota", AdminUpdateTenantQuota)

	// Admin API - backends
	server.RegisterRoute("GET", "/admin/api/v1/tenants/backends", AdminListBackends)
	server.RegisterRoute("POST", "/admin/api/v1/tenants/backends", AdminCreateBackend)
	server.RegisterRoute("PATCH", "/admin/api/v1/backends/update", AdminUpdateBackend)
	server.RegisterRoute("DELETE", "/admin/api/v1/backends/delete", AdminDeleteBackend)
}
