package middleware

import "github.com/bendy/file-gateway/internal/types"

// CORS adds cross-origin resource sharing headers.
func CORS() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			// Handle preflight
			if req.Method == "OPTIONS" {
				return types.Response{
					StatusCode: 204,
					Headers: map[string]string{
						"Access-Control-Allow-Origin":      "*",
						"Access-Control-Allow-Methods":     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
						"Access-Control-Allow-Headers":     "Authorization, Content-Type, X-Bendy-Timestamp",
						"Access-Control-Max-Age":           "86400",
						"Access-Control-Expose-Headers":    "Content-Length, Content-Type, Content-Disposition, ETag",
					},
				}
			}

			resp := next(req)

			// Add CORS headers to all responses
			if resp.Headers == nil {
				resp.Headers = map[string]string{}
			}
			resp.Headers["Access-Control-Allow-Origin"] = "*"
			resp.Headers["Access-Control-Expose-Headers"] = "Content-Length, Content-Type, Content-Disposition, ETag"

			return resp
		}
	}
}
