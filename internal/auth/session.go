package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bendy/file-gateway/internal/wasm"
)

const sessionDuration = 24 * time.Hour

// CreateAdminSession creates a new session for an admin user.
func CreateAdminSession(adminID string) (string, error) {
	sessionToken := generateToken(32)
	expiresAt := time.Now().Add(sessionDuration).UTC().Format(time.RFC3339)
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := wasm.DBExec(
		`INSERT INTO admin_sessions (id, admin_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		[]interface{}{sessionToken, adminID, expiresAt, createdAt},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionToken, nil
}

// ValidateAdminSession checks if a session token is valid.
func ValidateAdminSession(sessionToken string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Clean up expired sessions
	_, _ = wasm.DBExec(
		`DELETE FROM admin_sessions WHERE expires_at < ?`,
		[]interface{}{now},
	)

	rows, err := wasm.DBQuery(
		`SELECT s.admin_id, s.expires_at FROM admin_sessions s WHERE s.id = ?`,
		[]interface{}{sessionToken},
	)
	if err != nil || len(rows) == 0 {
		return "", fmt.Errorf("invalid session")
	}

	expiresAt, _ := time.Parse(time.RFC3339, rows[0]["expires_at"].(string))
	if time.Now().After(expiresAt) {
		return "", fmt.Errorf("session expired")
	}

	return rows[0]["admin_id"].(string), nil
}

// DeleteAdminSession removes an admin session (logout).
func DeleteAdminSession(sessionToken string) error {
	_, err := wasm.DBExec(
		`DELETE FROM admin_sessions WHERE id = ?`,
		[]interface{}{sessionToken},
	)
	return err
}

func generateToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
