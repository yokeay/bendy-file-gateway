package wasm

import (
	"encoding/json"
	"unsafe"
)

// Import declarations - functions provided by JS host

//go:wasmimport env dbQuery
func dbQuery(sqlPtr unsafe.Pointer, sqlLen int32, paramsPtr unsafe.Pointer, paramsLen int32) int64

//go:wasmimport env dbExec
func dbExec(sqlPtr unsafe.Pointer, sqlLen int32, paramsPtr unsafe.Pointer, paramsLen int32) int64

//go:wasmimport env cacheGet
func cacheGet(keyPtr unsafe.Pointer, keyLen int32) int64

//go:wasmimport env cacheSet
func cacheSet(keyPtr unsafe.Pointer, keyLen int32, valuePtr unsafe.Pointer, valueLen int32, ttlSeconds int32)

//go:wasmimport env cacheDel
func cacheDel(keyPtr unsafe.Pointer, keyLen int32)

// DBQuery executes a query via the JS host database.
func DBQuery(sql string, params []interface{}) ([]map[string]interface{}, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	sqlBytes := stringToBytes(sql)
	paramsBytes := paramsJSON

	resultPtr := dbQuery(
		unsafe.Pointer(&sqlBytes[0]), int32(len(sqlBytes)),
		unsafe.Pointer(&paramsBytes[0]), int32(len(paramsBytes)),
	)

	resultJSON := ptrToString(resultPtr)
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// DBExec executes a mutation via the JS host database.
func DBExec(sql string, params []interface{}) (int64, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}

	sqlBytes := stringToBytes(sql)
	paramsBytes := paramsJSON

	resultPtr := dbExec(
		unsafe.Pointer(&sqlBytes[0]), int32(len(sqlBytes)),
		unsafe.Pointer(&paramsBytes[0]), int32(len(paramsBytes)),
	)

	resultJSON := ptrToString(resultPtr)
	var result struct {
		RowsAffected  int64 `json:"rows_affected"`
		LastInsertID  int64 `json:"last_insert_id"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

// CacheGet retrieves a value from cache via JS host.
func CacheGet(key string) ([]byte, bool) {
	keyBytes := stringToBytes(key)
	resultPtr := cacheGet(unsafe.Pointer(&keyBytes[0]), int32(len(keyBytes)))

	result := ptrToBytes(resultPtr)
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

// CacheSet stores a value in cache via JS host.
func CacheSet(key string, value []byte, ttlSeconds int) {
	keyBytes := stringToBytes(key)
	cacheSet(
		unsafe.Pointer(&keyBytes[0]), int32(len(keyBytes)),
		unsafe.Pointer(&value[0]), int32(len(value)),
		int32(ttlSeconds),
	)
}

// CacheDel removes a key from cache via JS host.
func CacheDel(key string) {
	keyBytes := stringToBytes(key)
	cacheDel(unsafe.Pointer(&keyBytes[0]), int32(len(keyBytes)))
}

func stringToBytes(s string) []byte {
	return []byte(s)
}

func ptrToString(ptr int64) string {
	if ptr == 0 {
		return "[]"
	}
	// Read string from shared memory
	// In TinyGo WASM, pointers into linear memory work as int64
	return readStringFromMemory(uint32(ptr))
}

func ptrToBytes(ptr int64) []byte {
	if ptr == 0 {
		return nil
	}
	return readBytesFromMemory(uint32(ptr))
}

// Stub memory readers - these will be implemented with actual TinyGo memory ops
func readStringFromMemory(ptr uint32) string {
	buf := (*[1 << 16]byte)(unsafe.Pointer(uintptr(ptr)))
	length := 0
	for buf[length] != 0 {
		length++
	}
	return string(buf[:length])
}

func readBytesFromMemory(ptr uint32) []byte {
	buf := (*[1 << 16]byte)(unsafe.Pointer(uintptr(ptr)))
	length := 0
	for buf[length] != 0 {
		length++
	}
	result := make([]byte, length)
	copy(result, buf[:length])
	return result
}
