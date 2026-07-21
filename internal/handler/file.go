package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"github.com/bendy/file-gateway/internal/model"
	"github.com/bendy/file-gateway/internal/quota"
	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/types"
	"github.com/bendy/file-gateway/internal/util"
	"github.com/bendy/file-gateway/internal/wasm"
)

// UploadFile handles POST /api/v1/files/upload
func UploadFile(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	var body struct {
		VirtualName string `json:"virtual_name"`
		ContentType string `json:"content_type"`
		FileData    string `json:"file_data"`
		BackendID   string `json:"backend_id"`
		DirectoryID string `json:"directory_id"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}
	if body.VirtualName == "" || body.FileData == "" {
		return types.Error(400, "bad_request", "virtual_name and file_data are required", nil)
	}

	// Decode file data
	fileBytes, err := base64.StdEncoding.DecodeString(body.FileData)
	if err != nil {
		return types.Error(400, "bad_request", "invalid base64 file_data", nil)
	}

	// Check storage quota before uploading
	if err := quota.CheckStorageQuota(req.TenantID, int64(len(fileBytes))); err != nil {
		return types.Error(413, "storage_quota_exceeded", err.Error(), nil)
	}

	// Resolve backend
	backendID := body.BackendID
	if backendID == "" {
		rows, err := wasm.DBQuery(
			"SELECT id, driver, config FROM backends WHERE tenant_id = ? AND is_default = 1 AND status = 'active' LIMIT 1",
			[]interface{}{req.TenantID},
		)
		if err != nil || len(rows) == 0 {
			return types.Error(400, "no_backend", "no active backend configured for this tenant", nil)
		}
		backendID = asString(rows[0]["id"])
	}

	// Ensure backend is loaded in Manager
	if err := ensureBackendLoaded(backendID, req.TenantID); err != nil {
		return types.InternalError("backend operation failed")
	}

	// Build storage key
	contentType := body.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	storageKey := req.TenantID + "/" + util.UUID() + "/" + body.VirtualName

	// Upload to storage
	ctx := context.Background()
	info, err := storage.GetManager().Put(ctx, backendID, storageKey, bytes.NewReader(fileBytes), storage.UploadOptions{
		ContentType: contentType,
	})
	if err != nil {
		return types.InternalError("upload failed")
	}

	// Insert file record
	fileID := util.UUID()
	now := time.Now().UTC().Format(time.RFC3339)
	var dirID interface{} = nil
	if body.DirectoryID != "" {
		dirID = body.DirectoryID
	}

	_, err = wasm.DBExec(
		"INSERT INTO files (id, tenant_id, directory_id, backend_id, virtual_name, storage_key, content_type, size, checksum, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{fileID, req.TenantID, dirID, backendID, body.VirtualName, storageKey, contentType, info.Size, info.ETag, "{}", now, now},
	)
	if err != nil {
		// Rollback: delete from storage
		_ = storage.GetManager().Delete(ctx, backendID, storageKey)
		return types.Error(500, "db_error", "failed to save file record", nil)
	}

	// Track storage usage
	_ = quota.AdjustStorageUsed(req.TenantID, info.Size)

	return types.JSON(201, map[string]interface{}{
		"file": map[string]interface{}{
			"id":           fileID,
			"virtual_name": body.VirtualName,
			"storage_key":  storageKey,
			"content_type": contentType,
			"size":         info.Size,
			"checksum":     info.ETag,
			"backend_id":   backendID,
			"created_at":   now,
		},
	})
}

// DownloadFile handles GET /api/v1/files/download
func DownloadFile(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	fileID := params["id"]
	if fileID == "" {
		return types.Error(400, "bad_request", "file id is required", nil)
	}

	file, errResp := getFileRecord(fileID, req.TenantID)
	if errResp != nil {
		return *errResp
	}

	// Ensure backend loaded
	if err := ensureBackendLoaded(file.BackendID, req.TenantID); err != nil {
		return types.InternalError("backend operation failed")
	}

	ctx := context.Background()
	reader, info, err := storage.GetManager().Get(ctx, file.BackendID, file.StorageKey, storage.DownloadOptions{})
	if err != nil {
		return types.Error(404, "file_not_found", err.Error(), nil)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(reader)

	headers := map[string]string{
		"Content-Type":              info.ContentType,
		"Content-Disposition":       "attachment; filename=\"" + file.VirtualName + "\"",
		"Content-Length":            strconv.FormatInt(info.Size, 10),
		"X-Bendy-File-Name":         file.VirtualName,
		"X-Bendy-File-Checksum":     file.Checksum,
	}

	return types.Response{
		StatusCode: 200,
		Headers:    headers,
		Body:       buf.Bytes(),
	}
}

// FileInfo handles GET /api/v1/files/info
func FileInfo(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	fileID := params["id"]
	if fileID == "" {
		return types.Error(400, "bad_request", "file id is required", nil)
	}

	file, errResp := getFileRecord(fileID, req.TenantID)
	if errResp != nil {
		return *errResp
	}

	return types.JSON(200, map[string]interface{}{"file": file})
}

// ListFiles handles GET /api/v1/files/list
func ListFiles(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	dirID := params["directory_id"]

	var rows []map[string]interface{}
	var err error
	if dirID != "" {
		rows, err = wasm.DBQuery(
			"SELECT id, tenant_id, directory_id, backend_id, virtual_name, storage_key, content_type, size, checksum, metadata, created_at, updated_at FROM files WHERE tenant_id = ? AND directory_id = ? ORDER BY created_at DESC",
			[]interface{}{req.TenantID, dirID},
		)
	} else {
		rows, err = wasm.DBQuery(
			"SELECT id, tenant_id, directory_id, backend_id, virtual_name, storage_key, content_type, size, checksum, metadata, created_at, updated_at FROM files WHERE tenant_id = ? ORDER BY created_at DESC",
			[]interface{}{req.TenantID},
		)
	}
	if err != nil {
		return types.InternalError("database operation failed")
	}

	files := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		files = append(files, scanFileRow(row))
	}

	return types.JSON(200, map[string]interface{}{"files": files})
}

// DeleteFile handles DELETE /api/v1/files/delete
func DeleteFile(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	fileID := params["id"]
	if fileID == "" {
		return types.Error(400, "bad_request", "file id is required", nil)
	}

	file, errResp := getFileRecord(fileID, req.TenantID)
	if errResp != nil {
		return *errResp
	}

	// Delete from storage
	if err := ensureBackendLoaded(file.BackendID, req.TenantID); err != nil {
		return types.InternalError("backend operation failed")
	}
	ctx := context.Background()
	if err := storage.GetManager().Delete(ctx, file.BackendID, file.StorageKey); err != nil {
		// Log but continue — orphaned storage blob is better than inconsistent DB
	}

	// Delete from DB
	_, err := wasm.DBExec("DELETE FROM files WHERE id = ?", []interface{}{fileID})
	if err != nil {
		return types.InternalError("database operation failed")
	}

	// Release storage quota
	_ = quota.AdjustStorageUsed(req.TenantID, -file.Size)

	return types.JSON(200, map[string]interface{}{"deleted": true})
}

// getFileRecord fetches a file from DB and verifies tenant ownership.
func getFileRecord(fileID, tenantID string) (*model.File, *types.Response) {
	rows, err := wasm.DBQuery(
		"SELECT id, tenant_id, directory_id, backend_id, virtual_name, storage_key, content_type, size, checksum, metadata, created_at, updated_at FROM files WHERE id = ?",
		[]interface{}{fileID},
	)
	if err != nil || len(rows) == 0 {
		resp := types.Error(404, "not_found", "file not found", nil)
		return nil, &resp
	}

	file := scanFile(rows[0])
	if file.TenantID != tenantID {
		resp := types.Error(403, "forbidden", "file does not belong to this tenant", nil)
		return nil, &resp
	}
	return file, nil
}

// ensureBackendLoaded loads a backend from DB into the Manager if not already present.
func ensureBackendLoaded(backendID, tenantID string) error {
	rows, err := wasm.DBQuery(
		"SELECT id, driver, config FROM backends WHERE id = ? AND tenant_id = ? AND status = 'active'",
		[]interface{}{backendID, tenantID},
	)
	if err != nil || len(rows) == 0 {
		return err
	}

	driverName := asString(rows[0]["driver"])
	configJSON := asString(rows[0]["config"])
	cfg := map[string]string{}
	if configJSON != "" {
		json.Unmarshal([]byte(configJSON), &cfg)
	}

	return storage.GetManager().AddBackend(backendID, driverName, cfg)
}

// scanFile converts a DB row to a model.File.
func scanFile(row map[string]interface{}) *model.File {
	return &model.File{
		ID:          asString(row["id"]),
		TenantID:    asString(row["tenant_id"]),
		DirectoryID: nullableString(row["directory_id"]),
		BackendID:   asString(row["backend_id"]),
		VirtualName: asString(row["virtual_name"]),
		StorageKey:  asString(row["storage_key"]),
		ContentType: asString(row["content_type"]),
		Size:        asInt64(row["size"]),
		Checksum:    asString(row["checksum"]),
	}
}

// scanFileRow converts a DB row to a JSON-safe map.
func scanFileRow(row map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":           row["id"],
		"tenant_id":    row["tenant_id"],
		"directory_id": row["directory_id"],
		"backend_id":   row["backend_id"],
		"virtual_name": row["virtual_name"],
		"storage_key":  row["storage_key"],
		"content_type": row["content_type"],
		"size":         row["size"],
		"checksum":     row["checksum"],
		"created_at":   row["created_at"],
		"updated_at":   row["updated_at"],
	}
}

func nullableString(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := asString(v)
	if s == "" {
		return nil
	}
	return &s
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return ""
	}
}

func asInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(string(val), 10, 64)
		return n
	default:
		return 0
	}
}
