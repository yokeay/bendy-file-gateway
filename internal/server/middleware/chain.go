package middleware

import "github.com/bendy/file-gateway/internal/types"

// Chain composes middleware in order (first wraps second, etc.).
func Chain(middlewares ...types.Middleware) types.Middleware {
	return func(next types.Handler) types.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
