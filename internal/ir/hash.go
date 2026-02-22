package ir

import (
	"crypto/sha256"
	"fmt"
)

// ComputeHash computes the SHA-256 hash of the canonical JSON
// serialization of attributes (sorted keys, no whitespace).
func ComputeHash(attrs map[string]interface{}) string {
	data, err := SerializeCanonical(attrs)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}
