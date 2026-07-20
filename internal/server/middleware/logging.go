package middleware

import (
	"log"
	"time"

	"github.com/bendy/file-gateway/internal/server"
)

// Logging logs each request with method, path, status, and duration.
func Logging() server.Middleware {
	return func(next server.Handler) server.Handler {
		return func(req *server.Request) server.Response {
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
