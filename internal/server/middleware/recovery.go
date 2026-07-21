package middleware

import (
	"log"

	"github.com/bendy/file-gateway/internal/types"
)

// Recovery catches panics and returns a 500 error.
func Recovery() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) (resp types.Response) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic recovered: %v", r)
					resp = types.Error(500, "internal_error", "an internal error occurred", nil)
				}
			}()
			return next(req)
		}
	}
}
