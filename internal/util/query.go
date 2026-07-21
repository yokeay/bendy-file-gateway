package util

import "strings"

// QueryParams parses query string parameters from a URL path.
// e.g. "/api/v1/files/download?key=abc&name=hello" -> {"key": "abc", "name": "hello"}
func QueryParams(path string) map[string]string {
	result := map[string]string{}
	idx := strings.Index(path, "?")
	if idx < 0 {
		return result
	}
	qs := path[idx+1:]
	for _, pair := range strings.Split(qs, "&") {
		k, v, ok := strings.Cut(pair, "=")
		if ok {
			result[k] = v
		} else if k != "" {
			result[k] = ""
		}
	}
	return result
}
