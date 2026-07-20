package wasm

import (
	"encoding/json"
	"unsafe"
)

// RequestResult represents the processed HTTP response.
type RequestResult struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// HandleRequestFunc is the type for exported request handlers.
type HandleRequestFunc func(method, path, headersJSON, body string, remoteAddr string) RequestResult

var handleRequestFn HandleRequestFunc

var readyCalled bool

// ExportHandleRequest registers the request handler.
func ExportHandleRequest(fn HandleRequestFunc) {
	handleRequestFn = fn
}

// Ready signals to JS host that WASM is initialized.
func Ready() {
	readyCalled = true
}

//export handleRequest
func handleRequest(
	methodPtr unsafe.Pointer, methodLen int32,
	pathPtr unsafe.Pointer, pathLen int32,
	headersPtr unsafe.Pointer, headersLen int32,
	bodyPtr unsafe.Pointer, bodyLen int32,
	remoteAddrPtr unsafe.Pointer, remoteAddrLen int32,
) int64 {
	method := readString(methodPtr, methodLen)
	path := readString(pathPtr, pathLen)
	headersJSON := readString(headersPtr, headersLen)
	body := readString(bodyPtr, bodyLen)
	remoteAddr := readString(remoteAddrPtr, remoteAddrLen)

	result := handleRequestFn(method, path, headersJSON, body, remoteAddr)

	responseBytes, _ := json.Marshal(result)
	return writeToMemory(responseBytes)
}

//export ready
func ready() int32 {
	if readyCalled {
		return 1
	}
	return 0
}

func readString(ptr unsafe.Pointer, length int32) string {
	if ptr == nil || length <= 0 {
		return ""
	}
	buf := unsafe.Slice((*byte)(ptr), length)
	return string(buf)
}

func writeToMemory(data []byte) int64 {
	if len(data) == 0 {
		return 0
	}
	// Allocate memory that JS host can read
	buf := make([]byte, len(data)+1)
	copy(buf, data)
	buf[len(data)] = 0 // null terminator
	return int64(uintptr(unsafe.Pointer(&buf[0])))
}
