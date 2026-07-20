package main

import (
	"github.com/bendy/file-gateway/internal/config"
	"github.com/bendy/file-gateway/internal/server"
	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"

	// Register route handlers
	_ "github.com/bendy/file-gateway/internal/handler"

	// Register all storage drivers
	_ "github.com/bendy/file-gateway/internal/storage/drivers"
)

func main() {
	// Load config from JS host
	config.Init()

	// Initialize storage manager
	storage.Init()

	// Export handleRequest to JS host
	wasm.ExportHandleRequest(server.HandleRequest)

	// Signal ready to JS host
	wasm.Ready()

	// Keep alive
	select {}
}
