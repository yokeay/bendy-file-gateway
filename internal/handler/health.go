package handler

import "github.com/bendy/file-gateway/internal/server"

// Health is a public health check endpoint.
func Health(req *server.Request) server.Response {
	return server.JSON(200, map[string]interface{}{
		"status":  "ok",
		"version": "0.1.0",
		"service": "bendy-file-gateway",
	})
}
