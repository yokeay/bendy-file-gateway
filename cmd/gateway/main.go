package main

import (
	"unsafe"

	"github.com/bendy/file-gateway/internal/server"
	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"

	// Register all storage drivers
	_ "github.com/bendy/file-gateway/internal/storage/drivers"
)

func main() {
	// Initialize storage manager
	storage.Init()

	// Export handleRequest to JS host
	wasm.ExportHandleRequest(server.HandleRequest)

	// Signal ready to JS host
	wasm.Ready()

	// Keep alive
	select {}
}

//go:export malloc
func malloc(size uint32) unsafe.Pointer {
	buf := make([]byte, size)
	return unsafe.Pointer(&buf[0])
}

//go:export free
func free(ptr unsafe.Pointer, size uint32) {
	// GC will handle this
}
