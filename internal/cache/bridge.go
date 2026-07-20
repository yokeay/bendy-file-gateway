package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bendy/file-gateway/internal/wasm"
)

// Get retrieves a cached value and unmarshals it into the target.
// Returns false if the key doesn't exist.
func Get(key string, target interface{}) (bool, error) {
	raw, ok := wasm.CacheGet(key)
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, fmt.Errorf("cache unmarshal: %w", err)
	}
	return true, nil
}

// GetString retrieves a cached string value.
func GetString(key string) (string, bool) {
	raw, ok := wasm.CacheGet(key)
	if !ok {
		return "", false
	}
	return string(raw), true
}

// Set stores a value in cache with a TTL.
// Values with zero or negative TTL default to 60 seconds.
func Set(key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}

	wasm.CacheSet(key, data, int(ttl.Seconds()))
	return nil
}

// SetString stores a string value in cache with a TTL.
func SetString(key string, value string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	wasm.CacheSet(key, []byte(value), int(ttl.Seconds()))
}

// Del removes a key from cache.
func Del(key string) {
	wasm.CacheDel(key)
}
