package middleware

import "github.com/bendy/file-gateway/internal/types"

// SecurityHeaders adds common security-related HTTP headers.
func SecurityHeaders() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			resp := next(req)
			if resp.Headers == nil {
				resp.Headers = map[string]string{}
			}
			resp.Headers["X-Content-Type-Options"] = "nosniff"
			resp.Headers["X-Frame-Options"] = "DENY"
			resp.Headers["X-XSS-Protection"] = "1; mode=block"
			resp.Headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
			return resp
		}
	}
}
