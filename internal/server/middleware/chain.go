package middleware

import "github.com/bendy/file-gateway/internal/server"

// Chain composes middleware in order (first wraps second, etc.).
func Chain(middlewares ...server.Middleware) server.Middleware {
	return func(next server.Handler) server.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
