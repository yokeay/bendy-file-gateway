package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/auth"
	"github.com/bendy/file-gateway/internal/types"
	"github.com/bendy/file-gateway/internal/util"
	"github.com/bendy/file-gateway/internal/wasm"
)

type githubUserInput struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// AdminGitHubLogin handles POST /admin/api/v1/auth/github
func AdminGitHubLogin(req *types.Request) types.Response {
	var body struct {
		Code       string            `json:"code"`
		GitHubUser *githubUserInput  `json:"github_user"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}

	var ghLogin string
	var ghID int64
	var ghName string
	var ghAvatar string

	if body.GitHubUser != nil {
		// Pre-resolved by JS host (no fetch needed)
		ghLogin = body.GitHubUser.Login
		ghID = body.GitHubUser.ID
		ghName = body.GitHubUser.Name
		ghAvatar = body.GitHubUser.AvatarURL
	} else if body.Code != "" {
		// Exchange code via WASM fetch import
		client := auth.NewGitHubOAuthClient()
		token, err := client.ExchangeCode(body.Code)
		if err != nil {
			return types.Error(401, "auth_failed", err.Error(), nil)
		}
		ghUser, err := client.GetUser(token)
		if err != nil {
			return types.Error(401, "auth_failed", err.Error(), nil)
		}
		ghLogin = ghUser.Login
		ghID = ghUser.ID
		ghName = ghUser.Name
		ghAvatar = ghUser.AvatarURL
	} else {
		return types.Error(400, "bad_request", "code or github_user is required", nil)
	}

	client := auth.NewGitHubOAuthClient()
	if !client.IsAllowed(ghLogin) {
		return types.Error(403, "forbidden", "user not in admin list", nil)
	}

	// Upsert admin
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := wasm.DBQuery(
		"SELECT id FROM admins WHERE github_id = ?",
		[]interface{}{ghID},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	var adminID string
	if len(rows) > 0 {
		adminID = rows[0]["id"].(string)
		_, _ = wasm.DBExec(
			"UPDATE admins SET name = ?, avatar_url = ?, last_login_at = ?, updated_at = ? WHERE id = ?",
			[]interface{}{ghName, ghAvatar, now, now, adminID},
		)
	} else {
		adminID = util.UUID()
		_, err = wasm.DBExec(
			"INSERT INTO admins (id, github_username, github_id, name, avatar_url, role, last_login_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			[]interface{}{adminID, ghLogin, ghID, ghName, ghAvatar, "admin", now, now, now},
		)
		if err != nil {
			return types.Error(500, "internal_error", err.Error(), nil)
		}
	}

	sessionToken, err := auth.CreateAdminSession(adminID)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Set-Cookie":   "session_token=" + sessionToken + "; HttpOnly; SameSite=Lax; Path=/; Max-Age=86400",
		},
		Body: mustMarshal(map[string]interface{}{
			"admin": map[string]interface{}{
				"id":              adminID,
				"github_username": ghLogin,
				"name":            ghName,
				"avatar_url":      ghAvatar,
				"role":            "admin",
			},
		}),
	}
}

// AdminMe handles GET /admin/api/v1/auth/me
func AdminMe(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	rows, err := wasm.DBQuery(
		"SELECT id, github_username, github_id, name, avatar_url, role, last_login_at, created_at FROM admins WHERE id = ?",
		[]interface{}{req.AdminID},
	)
	if err != nil || len(rows) == 0 {
		return types.Error(404, "not_found", "admin not found", nil)
	}

	r := rows[0]
	return types.JSON(200, map[string]interface{}{
		"admin": map[string]interface{}{
			"id":              r["id"],
			"github_username": r["github_username"],
			"github_id":       r["github_id"],
			"name":            r["name"],
			"avatar_url":      r["avatar_url"],
			"role":            r["role"],
			"last_login_at":   r["last_login_at"],
			"created_at":      r["created_at"],
		},
	})
}

// AdminLogout handles POST /admin/api/v1/auth/logout
func AdminLogout(req *types.Request) types.Response {
	cookie := req.Headers["cookie"]
	if token := extractSessionToken(cookie); token != "" {
		_ = auth.DeleteAdminSession(token)
	}

	return types.JSON(200, map[string]interface{}{
		"message": "logged out",
	})
}

// AdminStats handles GET /admin/api/v1/stats
func AdminStats(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	tenantCount, _ := wasm.DBQuery("SELECT COUNT(*) as count FROM tenants", nil)
	fileCount, _ := wasm.DBQuery("SELECT COUNT(*) as count FROM files", nil)
	totalTraffic, _ := wasm.DBQuery("SELECT COALESCE(SUM(traffic_bytes), 0) as total FROM api_logs", nil)
	totalStorage, _ := wasm.DBQuery("SELECT COALESCE(SUM(storage_used), 0) as total FROM tenant_quotas", nil)

	getCount := func(rows []map[string]interface{}, err error) int64 {
		if err != nil || len(rows) == 0 {
			return 0
		}
		if v, ok := rows[0]["count"].(float64); ok {
			return int64(v)
		}
		if v, ok := rows[0]["total"].(float64); ok {
			return int64(v)
		}
		return 0
	}

	return types.JSON(200, map[string]interface{}{
		"stats": map[string]interface{}{
			"tenant_count":  getCount(tenantCount, nil),
			"file_count":    getCount(fileCount, nil),
			"total_traffic": getCount(totalTraffic, nil),
			"total_storage": getCount(totalStorage, nil),
		},
	})
}

// Tenant API - admin

// AdminListTenants handles GET /admin/api/v1/tenants
func AdminListTenants(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	rows, err := wasm.DBQuery(
		"SELECT id, name, access_key, status, created_at, updated_at FROM tenants ORDER BY created_at DESC",
		nil,
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	tenants := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		tenants = append(tenants, map[string]interface{}{
			"id":         r["id"],
			"name":       r["name"],
			"access_key": r["access_key"],
			"status":     r["status"],
			"created_at": r["created_at"],
			"updated_at": r["updated_at"],
		})
	}

	return types.JSON(200, map[string]interface{}{"tenants": tenants})
}

// AdminCreateTenant handles POST /admin/api/v1/tenants
func AdminCreateTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Name == "" {
		return types.Error(400, "bad_request", "name is required", nil)
	}

	id := util.UUID()
	accessKey := util.AccessKey()
	secretKey := util.AccessKey()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := wasm.DBExec(
		"INSERT INTO tenants (id, name, access_key, secret_key_hash, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{id, body.Name, accessKey, secretKey, "active", now, now},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	// Create default quota record
	quotaID := util.UUID()
	_, _ = wasm.DBExec(
		"INSERT INTO tenant_quotas (id, tenant_id, traffic_limit, traffic_used, api_calls_limit, api_calls_used, storage_limit, storage_used, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{quotaID, id, 0, 0, 0, 0, 0, 0, now, now},
	)

	return types.JSON(201, map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":         id,
			"name":       body.Name,
			"access_key": accessKey,
			"secret_key": secretKey,
			"status":     "active",
			"created_at": now,
		},
	})
}

// AdminGetTenant handles GET /admin/api/v1/tenants/detail
func AdminGetTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	rows, err := wasm.DBQuery(
		"SELECT id, name, access_key, status, created_at, updated_at FROM tenants WHERE id = ?",
		[]interface{}{id},
	)
	if err != nil || len(rows) == 0 {
		return types.Error(404, "not_found", "tenant not found", nil)
	}

	r := rows[0]
	return types.JSON(200, map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":         r["id"],
			"name":       r["name"],
			"access_key": r["access_key"],
			"status":     r["status"],
			"created_at": r["created_at"],
			"updated_at": r["updated_at"],
		},
	})
}

// AdminUpdateTenant handles PATCH /admin/api/v1/tenants/update
func AdminUpdateTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	var body struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if body.Name != nil {
		_, err := wasm.DBExec(
			"UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?",
			[]interface{}{*body.Name, now, id},
		)
		if err != nil {
			return types.Error(500, "internal_error", err.Error(), nil)
		}
	}

	if body.Status != nil {
		_, err := wasm.DBExec(
			"UPDATE tenants SET status = ?, updated_at = ? WHERE id = ?",
			[]interface{}{*body.Status, now, id},
		)
		if err != nil {
			return types.Error(500, "internal_error", err.Error(), nil)
		}
	}

	return types.JSON(200, map[string]interface{}{"message": "updated"})
}

// AdminDeleteTenant handles DELETE /admin/api/v1/tenants/delete
func AdminDeleteTenant(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	_, err := wasm.DBExec(
		"DELETE FROM tenants WHERE id = ?",
		[]interface{}{id},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.JSON(200, map[string]interface{}{"message": "deleted"})
}

// AdminRotateKey handles POST /admin/api/v1/tenants/rotate-key
func AdminRotateKey(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	newSecret := util.AccessKey()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := wasm.DBExec(
		"UPDATE tenants SET secret_key_hash = ?, updated_at = ? WHERE id = ?",
		[]interface{}{newSecret, now, id},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.JSON(200, map[string]interface{}{
		"secret_key": newSecret,
	})
}

// Quota admin handlers

// AdminGetTenantQuota handles GET /admin/api/v1/tenants/quota
func AdminGetTenantQuota(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	tenantID := params["tenant_id"]
	if tenantID == "" {
		return types.Error(400, "bad_request", "tenant_id parameter is required", nil)
	}

	rows, err := wasm.DBQuery(
		"SELECT id, tenant_id, traffic_limit, traffic_used, api_calls_limit, api_calls_used, storage_limit, storage_used, expiry_at, created_at, updated_at FROM tenant_quotas WHERE tenant_id = ?",
		[]interface{}{tenantID},
	)
	if err != nil || len(rows) == 0 {
		return types.Error(404, "not_found", "quota not found", nil)
	}

	r := rows[0]
	return types.JSON(200, map[string]interface{}{
		"quota": map[string]interface{}{
			"id":              r["id"],
			"tenant_id":       r["tenant_id"],
			"traffic_limit":   r["traffic_limit"],
			"traffic_used":    r["traffic_used"],
			"api_calls_limit": r["api_calls_limit"],
			"api_calls_used":  r["api_calls_used"],
			"storage_limit":   r["storage_limit"],
			"storage_used":    r["storage_used"],
			"expiry_at":       r["expiry_at"],
			"created_at":      r["created_at"],
			"updated_at":      r["updated_at"],
		},
	})
}

// AdminUpdateTenantQuota handles PATCH /admin/api/v1/tenants/quota
func AdminUpdateTenantQuota(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	tenantID := params["tenant_id"]
	if tenantID == "" {
		return types.Error(400, "bad_request", "tenant_id parameter is required", nil)
	}

	var body struct {
		TrafficLimit  *int64  `json:"traffic_limit"`
		APICallsLimit *int64  `json:"api_calls_limit"`
		StorageLimit  *int64  `json:"storage_limit"`
		ExpiryAt      *string `json:"expiry_at"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if body.TrafficLimit != nil {
		_, _ = wasm.DBExec("UPDATE tenant_quotas SET traffic_limit = ?, updated_at = ? WHERE tenant_id = ?",
			[]interface{}{*body.TrafficLimit, now, tenantID})
	}
	if body.APICallsLimit != nil {
		_, _ = wasm.DBExec("UPDATE tenant_quotas SET api_calls_limit = ?, updated_at = ? WHERE tenant_id = ?",
			[]interface{}{*body.APICallsLimit, now, tenantID})
	}
	if body.StorageLimit != nil {
		_, _ = wasm.DBExec("UPDATE tenant_quotas SET storage_limit = ?, updated_at = ? WHERE tenant_id = ?",
			[]interface{}{*body.StorageLimit, now, tenantID})
	}
	if body.ExpiryAt != nil {
		_, _ = wasm.DBExec("UPDATE tenant_quotas SET expiry_at = ?, updated_at = ? WHERE tenant_id = ?",
			[]interface{}{*body.ExpiryAt, now, tenantID})
	}

	return types.JSON(200, map[string]interface{}{"message": "updated"})
}

// Backend admin handlers

// AdminListBackends handles GET /admin/api/v1/tenants/backends
func AdminListBackends(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	tenantID := params["tenant_id"]
	if tenantID == "" {
		return types.Error(400, "bad_request", "tenant_id parameter is required", nil)
	}

	rows, err := wasm.DBQuery(
		"SELECT id, tenant_id, name, driver, config, is_default, status, created_at, updated_at FROM backends WHERE tenant_id = ? ORDER BY created_at DESC",
		[]interface{}{tenantID},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	backends := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		var configMap map[string]interface{}
		if configStr, ok := r["config"].(string); ok {
			json.Unmarshal([]byte(configStr), &configMap)
		}
		backends = append(backends, map[string]interface{}{
			"id":         r["id"],
			"tenant_id":  r["tenant_id"],
			"name":       r["name"],
			"driver":     r["driver"],
			"config":     configMap,
			"is_default": r["is_default"],
			"status":     r["status"],
			"created_at": r["created_at"],
			"updated_at": r["updated_at"],
		})
	}

	return types.JSON(200, map[string]interface{}{"backends": backends})
}

// AdminCreateBackend handles POST /admin/api/v1/tenants/backends
func AdminCreateBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	var body struct {
		TenantID  string                 `json:"tenant_id"`
		Name      string                 `json:"name"`
		Driver    string                 `json:"driver"`
		Config    map[string]interface{} `json:"config"`
		IsDefault bool                   `json:"is_default"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Name == "" || body.Driver == "" || body.TenantID == "" {
		return types.Error(400, "bad_request", "tenant_id, name, and driver are required", nil)
	}

	configJSON, _ := json.Marshal(body.Config)
	id := util.UUID()
	now := time.Now().UTC().Format(time.RFC3339)
	isDefault := 0
	if body.IsDefault {
		isDefault = 1
	}

	_, err := wasm.DBExec(
		"INSERT INTO backends (id, tenant_id, name, driver, config, is_default, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{id, body.TenantID, body.Name, body.Driver, string(configJSON), isDefault, "active", now, now},
	)
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.JSON(201, map[string]interface{}{
		"backend": map[string]interface{}{
			"id":         id,
			"tenant_id":  body.TenantID,
			"name":       body.Name,
			"driver":     body.Driver,
			"config":     body.Config,
			"is_default": isDefault,
			"status":     "active",
			"created_at": now,
		},
	})
}

// AdminUpdateBackend handles PATCH /admin/api/v1/backends/update
func AdminUpdateBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	var body struct {
		Name      *string                 `json:"name"`
		Config    *map[string]interface{} `json:"config"`
		Status    *string                 `json:"status"`
		IsDefault *bool                   `json:"is_default"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return types.Error(400, "bad_request", "invalid JSON body", nil)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if body.Name != nil {
		_, _ = wasm.DBExec("UPDATE backends SET name = ?, updated_at = ? WHERE id = ?",
			[]interface{}{*body.Name, now, id})
	}
	if body.Config != nil {
		configJSON, _ := json.Marshal(*body.Config)
		_, _ = wasm.DBExec("UPDATE backends SET config = ?, updated_at = ? WHERE id = ?",
			[]interface{}{string(configJSON), now, id})
	}
	if body.Status != nil {
		_, _ = wasm.DBExec("UPDATE backends SET status = ?, updated_at = ? WHERE id = ?",
			[]interface{}{*body.Status, now, id})
	}
	if body.IsDefault != nil {
		isDef := 0
		if *body.IsDefault {
			isDef = 1
		}
		_, _ = wasm.DBExec("UPDATE backends SET is_default = ?, updated_at = ? WHERE id = ?",
			[]interface{}{isDef, now, id})
	}

	return types.JSON(200, map[string]interface{}{"message": "updated"})
}

// AdminDeleteBackend handles DELETE /admin/api/v1/backends/delete
func AdminDeleteBackend(req *types.Request) types.Response {
	if !req.IsAdmin {
		return types.Error(401, "unauthorized", "admin authentication required", nil)
	}

	params := util.QueryParams(req.Path)
	id := params["id"]
	if id == "" {
		return types.Error(400, "bad_request", "id parameter is required", nil)
	}

	_, err := wasm.DBExec("DELETE FROM backends WHERE id = ?", []interface{}{id})
	if err != nil {
		return types.Error(500, "internal_error", err.Error(), nil)
	}

	return types.JSON(200, map[string]interface{}{"message": "deleted"})
}

func extractSessionToken(cookie string) string {
	for _, c := range splitCookies(cookie) {
		c = strings.TrimSpace(c)
		parts := strings.SplitN(c, "=", 2)
		if len(parts) == 2 && parts[0] == "session_token" {
			return parts[1]
		}
	}
	return ""
}

func splitCookies(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
