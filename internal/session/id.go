package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateSecureID creates a cryptographically random session ID with at least
// 128 bits of entropy. The ID is prefixed with "sess_" and uses URL-safe
// base64 encoding (no padding) for the random component.
func generateSecureID() string {
	b := make([]byte, 16) // 128 bits
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return "sess_" + base64.RawURLEncoding.EncodeToString(b)
}
