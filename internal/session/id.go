package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateID creates a cryptographically random ID with at least 128 bits of entropy.
// The ID uses the given prefix and URL-safe base64 encoding (no padding).
func GenerateID(prefix string) string {
	b := make([]byte, 16) // 128 bits
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

// generateSecureID creates a session ID with "sess_" prefix for backward compatibility.
func generateSecureID() string {
	return GenerateID("sess_")
}
