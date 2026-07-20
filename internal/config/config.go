package config

import "github.com/bendy/file-gateway/internal/wasm"

var values = map[string]string{}

// Set stores a configuration value.
func Set(key, value string) {
	values[key] = value
}

// Get retrieves a configuration value.
func Get(key string) string {
	return values[key]
}

// Init loads configuration from the JS host via WASM imports.
func Init() {
	keys := []string{
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"ADMIN_GITHUB_USERNAMES",
		"GITHUB_REDIRECT_URI",
		"SESSION_SECRET",
		"VERSION",
	}
	for _, k := range keys {
		if v := wasm.GetEnv(k); v != "" {
			Set(k, v)
		}
	}
}
