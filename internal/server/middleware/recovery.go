package middleware

import (
	"log"

	"github.com/bendy/file-gateway/internal/server"
)

// Recovery catches panics and returns a 500 error.
func Recovery() server.Middleware {
	return func(next server.Handler) server.Handler {
		return func(req *server.Request) server.Response {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic recovered: %v", r)
				}
			}()
			return next(req)
		}
	}
}
