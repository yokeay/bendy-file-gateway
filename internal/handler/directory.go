package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bendy/file-gateway/internal/types"
	"github.com/bendy/file-gateway/internal/util"
	"github.com/bendy/file-gateway/internal/wasm"
)

func scanDir(row map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":         row["id"],
		"tenant_id":  row["tenant_id"],
		"parent_id":  row["parent_id"],
		"name":       row["name"],
		"path":       row["path"],
		"created_at": row["created_at"],
		"updated_at": row["updated_at"],
	}
}

// CreateDirectory handles POST /api/v1/directories
func CreateDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	var body struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}
	if body.Name == "" {
		return types.Error(400, "bad_request", "name is required", nil)
	}

	// Sanitize: strip slashes from name
	for _, c := range body.Name {
		if c == '/' || c == '\\' {
			return types.Error(400, "bad_request", "directory name must not contain slashes", nil)
		}
	}

	// Compute path
	dirPath := "/" + body.Name
	if body.ParentID != nil && *body.ParentID != "" {
		rows, err := wasm.DBQuery(
			"SELECT id, path FROM directories WHERE id = ? AND tenant_id = ?",
			[]interface{}{*body.ParentID, req.TenantID},
		)
		if err != nil || len(rows) == 0 {
			return types.Error(404, "not_found", "parent directory not found", nil)
		}
		parentPath := asString(rows[0]["path"])
		dirPath = parentPath + "/" + body.Name
	}

	dirID := util.UUID()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := wasm.DBExec(
		"INSERT INTO directories (id, tenant_id, parent_id, name, path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{dirID, req.TenantID, body.ParentID, body.Name, dirPath, now, now},
	)
	if err != nil {
		return types.InternalError("database operation failed")
	}

	return types.JSON(201, map[string]interface{}{
		"directory": map[string]interface{}{
			"id":         dirID,
			"tenant_id":  req.TenantID,
			"parent_id":  body.ParentID,
			"name":       body.Name,
			"path":       dirPath,
			"created_at": now,
			"updated_at": now,
		},
	})
}

// ListDirectory handles GET /api/v1/directories
func ListDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	parentID := params["parent_id"]

	var rows []map[string]interface{}
	var err error
	if parentID != "" {
		rows, err = wasm.DBQuery(
			"SELECT id, tenant_id, parent_id, name, path, created_at, updated_at FROM directories WHERE tenant_id = ? AND parent_id = ? ORDER BY name ASC",
			[]interface{}{req.TenantID, parentID},
		)
	} else {
		rows, err = wasm.DBQuery(
			"SELECT id, tenant_id, parent_id, name, path, created_at, updated_at FROM directories WHERE tenant_id = ? AND parent_id IS NULL ORDER BY name ASC",
			[]interface{}{req.TenantID},
		)
	}
	if err != nil {
		return types.InternalError("database operation failed")
	}

	dirs := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		dirs = append(dirs, scanDir(row))
	}

	return types.JSON(200, map[string]interface{}{"directories": dirs})
}

// DeleteDirectory handles DELETE /api/v1/directories
func DeleteDirectory(req *types.Request) types.Response {
	if req.TenantID == "" {
		return types.Error(401, "unauthorized", "tenant authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	dirID := params["id"]
	if dirID == "" {
		return types.Error(400, "bad_request", "directory id is required", nil)
	}

	// Verify tenant owns this directory
	rows, err := wasm.DBQuery(
		"SELECT id FROM directories WHERE id = ? AND tenant_id = ?",
		[]interface{}{dirID, req.TenantID},
	)
	if err != nil || len(rows) == 0 {
		return types.Error(404, "not_found", "directory not found", nil)
	}

	childCount, err := wasm.DBQuery(
		"SELECT COUNT(*) as count FROM directories WHERE parent_id = ?",
		[]interface{}{dirID},
	)
	if err == nil && len(childCount) > 0 {
		if c, ok := childCount[0]["count"].(float64); ok && c > 0 {
			return types.Error(409, "not_empty",
				fmt.Sprintf("directory has %d children, delete them first", int(c)), nil)
		}
	}

	fileCount, err := wasm.DBQuery(
		"SELECT COUNT(*) as count FROM files WHERE directory_id = ?",
		[]interface{}{dirID},
	)
	if err == nil && len(fileCount) > 0 {
		if c, ok := fileCount[0]["count"].(float64); ok && c > 0 {
			return types.Error(409, "not_empty",
				fmt.Sprintf("directory has %d files, move or delete them first", int(c)), nil)
		}
	}

	_, err = wasm.DBExec("DELETE FROM directories WHERE id = ?", []interface{}{dirID})
	if err != nil {
		return types.InternalError("database operation failed")
	}

	return types.JSON(200, map[string]interface{}{"deleted": true})
}
