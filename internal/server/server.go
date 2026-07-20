package server

import (
	"encoding/json"
	"log"

	"github.com/bendy/file-gateway/internal/server/middleware"
	"github.com/bendy/file-gateway/internal/types"
	"github.com/bendy/file-gateway/internal/wasm"
)

// HandleRequest is the main entry point for all HTTP requests.
// It is called from the WASM host (JS) via exports.
func HandleRequest(method, path, headersJSON, body, remoteAddr string) wasm.RequestResult {
	req := &types.Request{
		Method:     method,
		Path:       path,
		Headers:    parseHeaders(headersJSON),
		Body:       body,
		RemoteAddr: remoteAddr,
	}

	// Build middleware chain
	handler := middleware.Chain(
		middleware.Recovery(),
		middleware.CORS(),
		middleware.Logging(),
		middleware.Auth(),
		middleware.Quota(),
	)(router())

	resp := handler(req)

	return wasm.RequestResult{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       string(resp.Body),
	}
}

func parseHeaders(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		log.Printf("failed to parse headers: %v", err)
		return map[string]string{}
	}
	return headers
}
