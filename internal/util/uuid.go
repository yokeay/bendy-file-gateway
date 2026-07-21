package util

import (
	"crypto/rand"
	"encoding/hex"
)

// UUID generates a random 16-byte hex-encoded UUID.
func UUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return hex.EncodeToString(b)
}

// AccessKey generates a random 20-byte access key.
func AccessKey() string {
	b := make([]byte, 20)
	rand.Read(b)
	return hex.EncodeToString(b)
}
