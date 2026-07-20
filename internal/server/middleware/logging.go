package middleware

import (
	"log"
	"time"

	"github.com/bendy/file-gateway/internal/types"
)

// Logging logs each request with method, path, status, and duration.
func Logging() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			start := time.Now()
			resp := next(req)
			duration := time.Since(start)

			log.Printf("[%s] %s %s - %d (%v)",
				time.Now().Format(time.RFC3339),
				req.Method,
				req.Path,
				resp.StatusCode,
				duration,
			)

			return resp
		}
	}
}
