package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/config"
	"github.com/bendy/file-gateway/internal/wasm"
)

const sessionDuration = 24 * time.Hour

// CreateAdminSession creates a new session for an admin user.
func CreateAdminSession(adminID string) (string, error) {
	sessionToken := generateToken(32)
	expiresAt := time.Now().Add(sessionDuration).UTC().Format(time.RFC3339)
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := wasm.DBExec(
		`INSERT INTO admin_sessions (id, admin_id, session_token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		[]interface{}{sessionToken, adminID, sessionToken, expiresAt, createdAt},
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

// CreateSignedToken creates a signed session token (HMAC-SHA256 format: hexSig.adminID.expiryUnix).
// This is used by the JS host's OAuth callback handler to create sessions without D1 dependency.
func CreateSignedToken(adminID string) (string, error) {
	secret := config.Get("SESSION_SECRET")
	if secret == "" {
		return "", fmt.Errorf("SESSION_SECRET not configured")
	}

	expiresAtUnix := time.Now().Add(sessionDuration).Unix()
	payload := adminID + "." + strconv.FormatInt(expiresAtUnix, 10)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sigHex := hex.EncodeToString(mac.Sum(nil))

	return sigHex + "." + adminID + "." + strconv.FormatInt(expiresAtUnix, 10), nil
}

// VerifySignedToken validates a signed session token (HMAC-SHA256 format: hexSig.adminID.expiry).
// This is used by the auth middleware to validate sessions without D1 dependency.
func VerifySignedToken(token string) (string, error) {
	secret := config.Get("SESSION_SECRET")
	if secret == "" {
		return "", fmt.Errorf("SESSION_SECRET not configured")
	}

	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid session token format")
	}

	sigHex := parts[0]
	adminID := parts[1]
	expiresAtStr := parts[2]

	expiresAtUnix, err := strconv.ParseInt(expiresAtStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid expiry in session token")
	}

	if time.Now().Unix() > expiresAtUnix {
		return "", fmt.Errorf("session expired")
	}

	payload := adminID + "." + expiresAtStr
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedHex := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigHex), []byte(expectedHex)) {
		return "", fmt.Errorf("invalid session signature")
	}

	return adminID, nil
}

func generateToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
