package quota

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bendy/file-gateway/internal/wasm"
)

// QuotaInfo represents a tenant's usage and limits.
type QuotaInfo struct {
	TenantID      string     `json:"tenant_id"`
	TrafficLimit  int64      `json:"traffic_limit"`
	TrafficUsed   int64      `json:"traffic_used"`
	APICallsLimit int64      `json:"api_calls_limit"`
	APICallsUsed  int64      `json:"api_calls_used"`
	ExpiryAt      *time.Time `json:"expiry_at"`
}

// GetQuota retrieves quota information for a tenant.
// Checks cache first, then falls back to database.
func GetQuota(tenantID string) (*QuotaInfo, error) {
	cacheKey := "quota:" + tenantID

	// Try cache
	if cached, ok := wasm.CacheGet(cacheKey); ok {
		var q QuotaInfo
		if err := json.Unmarshal(cached, &q); err == nil {
			return &q, nil
		}
	}

	// Query database
	rows, err := wasm.DBQuery(
		`SELECT id, tenant_id, traffic_limit, traffic_used, api_calls_limit, api_calls_used, expiry_at
		 FROM tenant_quotas WHERE tenant_id = ?`,
		[]interface{}{tenantID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query quota: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no quota found for tenant")
	}

	q := &QuotaInfo{
		TenantID:      getString(rows[0], "tenant_id"),
		TrafficLimit:  getInt64(rows[0], "traffic_limit"),
		TrafficUsed:   getInt64(rows[0], "traffic_used"),
		APICallsLimit: getInt64(rows[0], "api_calls_limit"),
		APICallsUsed:  getInt64(rows[0], "api_calls_used"),
	}

	if expStr := getString(rows[0], "expiry_at"); expStr != "" && expStr != "null" {
		t, err := time.Parse(time.RFC3339, expStr)
		if err == nil {
			q.ExpiryAt = &t
		}
	}

	// Populate cache
	if data, err := json.Marshal(q); err == nil {
		wasm.CacheSet(cacheKey, data, 60)
	}

	return q, nil
}

// DeductQuota atomically updates quota usage counters.
func DeductQuota(tenantID string, callCount int64, bytesTransferred int64) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Atomic update that won't exceed limits
	rowsAffected, err := wasm.DBExec(
		`UPDATE tenant_quotas
		 SET api_calls_used = api_calls_used + ?,
		     traffic_used = traffic_used + ?,
		     updated_at = ?
		 WHERE tenant_id = ?
		 AND (api_calls_limit = 0 OR api_calls_used + ? <= api_calls_limit)
		 AND (traffic_limit = 0 OR traffic_used + ? <= traffic_limit)`,
		[]interface{}{callCount, bytesTransferred, now, tenantID, callCount, bytesTransferred},
	)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("quota exceeded")
	}

	// Invalidate cache
	wasm.CacheDel("quota:" + tenantID)

	return nil
}

// LogAPIRequest records an API request to the api_logs table for billing/audit.
func LogAPIRequest(tenantID, method, path, remoteAddr string, statusCode, trafficBytes, durationMs int64) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("log-%d", time.Now().UnixNano())
	_, _ = wasm.DBExec(
		"INSERT INTO api_logs (id, tenant_id, method, path, status_code, traffic_bytes, duration_ms, remote_addr, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		[]interface{}{id, tenantID, method, path, statusCode, trafficBytes, durationMs, remoteAddr, now},
	)
}

// CheckStorageQuota verifies the tenant has room for additional bytes.
// Returns nil if within limits, or an error if the limit would be exceeded.
func CheckStorageQuota(tenantID string, additionalBytes int64) error {
	rows, err := wasm.DBQuery(
		"SELECT storage_limit, storage_used FROM tenant_quotas WHERE tenant_id = ?",
		[]interface{}{tenantID},
	)
	if err != nil || len(rows) == 0 {
		return fmt.Errorf("failed to query storage quota")
	}
	limit := getInt64(rows[0], "storage_limit")
	used := getInt64(rows[0], "storage_used")

	if limit > 0 && used+additionalBytes > limit {
		return fmt.Errorf("storage limit exceeded: %d/%d bytes used", used, limit)
	}
	return nil
}

// AdjustStorageUsed atomically updates the tenant's storage_used counter.
func AdjustStorageUsed(tenantID string, delta int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	rowsAffected, err := wasm.DBExec(
		`UPDATE tenant_quotas
		 SET storage_used = storage_used + ?,
		     updated_at = ?
		 WHERE tenant_id = ?
		 AND (storage_limit = 0 OR storage_used + ? <= storage_limit)`,
		[]interface{}{delta, now, tenantID, delta},
	)
	if err != nil {
		return fmt.Errorf("failed to update storage quota: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("storage quota update rejected: limit would be exceeded")
	}
	wasm.CacheDel("quota:" + tenantID)
	return nil
}

// ResetQuotaCache invalidates the cached quota for a tenant.
func ResetQuotaCache(tenantID string) {
	wasm.CacheDel("quota:" + tenantID)
}

func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getInt64(row map[string]interface{}, key string) int64 {
	s := getString(row, key)
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
