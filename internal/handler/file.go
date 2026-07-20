package handler

import "github.com/bendy/file-gateway/internal/server"

// UploadFile handles POST /api/v1/files/upload
func UploadFile(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "upload not yet implemented", nil)
}

// DownloadFile handles GET /api/v1/files/download
func DownloadFile(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "download not yet implemented", nil)
}

// FileInfo handles GET /api/v1/files/info
func FileInfo(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "file info not yet implemented", nil)
}

// ListFiles handles GET /api/v1/files/list
func ListFiles(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "list files not yet implemented", nil)
}

// DeleteFile handles DELETE /api/v1/files/delete
func DeleteFile(req *server.Request) server.Response {
	if req.TenantID == "" {
		return server.Error(401, "unauthorized", "tenant authentication required", nil)
	}
	return server.Error(501, "not_implemented", "delete file not yet implemented", nil)
}
