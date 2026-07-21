package middleware

import (
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/quota"
	"github.com/bendy/file-gateway/internal/types"
)

// Quota enforces tenant quota limits on API requests.
func Quota() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			// Only check quota for tenant API requests
			if !strings.HasPrefix(req.Path, "/api/") || req.TenantID == "" {
				return next(req)
			}

			// Skip quota check for quota info endpoint itself
			if strings.HasSuffix(req.Path, "/quota") && req.Method == "GET" {
				return next(req)
			}

			q, err := quota.GetQuota(req.TenantID)
			if err != nil {
				return types.Error(500, "internal_error", "failed to check quota", nil)
			}

			// Check expiry
			if q.ExpiryAt != nil && time.Now().After(*q.ExpiryAt) {
				return types.Error(403, "forbidden", "access key has expired", map[string]interface{}{
					"expired_at": q.ExpiryAt.Format(time.RFC3339),
				})
			}

			// Check API calls limit
			if q.APICallsLimit > 0 && q.APICallsUsed >= q.APICallsLimit {
				return types.Error(429, "quota_exceeded", "API calls limit exceeded", map[string]interface{}{
					"limit": q.APICallsLimit,
					"used":  q.APICallsUsed,
				})
			}

			// Check traffic limit
			if q.TrafficLimit > 0 && q.TrafficUsed >= q.TrafficLimit {
				return types.Error(429, "quota_exceeded", "traffic limit exceeded", map[string]interface{}{
					"limit": q.TrafficLimit,
					"used":  q.TrafficUsed,
				})
			}

			// Store quota in request context for post-request update
			req.QuotaData = q

			// Execute handler
			start := time.Now()
			resp := next(req)
			duration := time.Since(start)

			// Post-request: update quota usage
			bytesTransferred := int64(len(req.Body) + len(resp.Body))
			if err := quota.DeductQuota(req.TenantID, 1, bytesTransferred); err != nil {
				// Log but don't fail - the request succeeded
			}

			// Log API request for billing/audit
			quota.LogAPIRequest(
				req.TenantID, req.Method, req.Path, req.RemoteAddr,
				int64(resp.StatusCode), bytesTransferred, duration.Milliseconds(),
			)

			return resp
		}
	}
}
