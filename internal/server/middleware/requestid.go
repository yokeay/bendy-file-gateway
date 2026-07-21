package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bendy/file-gateway/internal/types"
)

// RequestID generates a short request ID and attaches it to the request.
func RequestID() types.Middleware {
	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			req.RequestID = shortID()
			resp := next(req)
			if resp.Headers == nil {
				resp.Headers = map[string]string{}
			}
			resp.Headers["X-Request-ID"] = req.RequestID
			return resp
		}
	}
}

func shortID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
