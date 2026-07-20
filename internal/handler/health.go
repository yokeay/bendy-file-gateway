package handler

import "github.com/bendy/file-gateway/internal/types"

// Health is a public health check endpoint.
func Health(req *types.Request) types.Response {
	return types.JSON(200, map[string]interface{}{
		"status":  "ok",
		"version": "0.1.0",
		"service": "bendy-file-gateway",
	})
}
