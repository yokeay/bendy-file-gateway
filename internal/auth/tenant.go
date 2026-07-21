package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/types"
	"github.com/bendy/file-gateway/internal/wasm"
)

const maxClockSkew = 5 * time.Minute

// TenantAuthResult holds the parsed tenant identity.
type TenantAuthResult struct {
	TenantID  string
	AccessKey string
}

// VerifyTenantRequest verifies the HMAC-SHA256 signature on a tenant API request.
func VerifyTenantRequest(req *types.Request) (*TenantAuthResult, error) {
	authHeader := req.Headers["authorization"]
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	scheme, params, ok := strings.Cut(authHeader, " ")
	if !ok || scheme != "HMAC-SHA256" {
		return nil, fmt.Errorf("invalid auth scheme")
	}

	accessKey, clientSig, ok := strings.Cut(params, ":")
	if !ok {
		return nil, fmt.Errorf("invalid auth format, expected AccessKey:Signature")
	}

	// Look up tenant by access key
	rows, err := wasm.DBQuery(
		"SELECT id, secret_key_hash FROM tenants WHERE access_key = ? AND status = 'active'",
		[]interface{}{accessKey},
	)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("invalid credentials")
	}

	tenantID := rows[0]["id"].(string)
	secretHash := rows[0]["secret_key_hash"].(string)

	// Verify timestamp
	tsHeader := req.Headers["x-bendy-timestamp"]
	if tsHeader == "" {
		return nil, fmt.Errorf("missing X-Bendy-Timestamp header")
	}

	reqTime, err := time.Parse(time.RFC3339, tsHeader)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format")
	}

	now := time.Now()
	if now.Sub(reqTime) > maxClockSkew || reqTime.Sub(now) > maxClockSkew {
		return nil, fmt.Errorf("request timestamp expired")
	}

	// Compute content SHA256
	bodyHash := sha256.Sum256([]byte(req.Body))
	contentSHA256 := hex.EncodeToString(bodyHash[:])

	// Build string-to-sign
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		req.Method,
		req.Path,
		tsHeader,
		contentSHA256,
	)

	// Verify HMAC
	expectedSig := computeHMAC(secretHash, stringToSign)
	if !hmac.Equal([]byte(expectedSig), []byte(clientSig)) {
		return nil, fmt.Errorf("signature mismatch")
	}

	return &TenantAuthResult{
		TenantID:  tenantID,
		AccessKey: accessKey,
	}, nil
}

func computeHMAC(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// HashSecret returns the SHA-256 hash of a secret key for storage.
func HashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}
