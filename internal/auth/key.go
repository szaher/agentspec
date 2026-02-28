// Package auth provides API key validation and HTTP authentication
// middleware for compiled agents and the runtime server.
package auth

import (
	"crypto/subtle"
	"os"
)

// DefaultEnvVar is the environment variable name for the API key.
const DefaultEnvVar = "AGENTSPEC_API_KEY"

// ValidateKey performs timing-safe comparison of the provided key
// against the expected key. Returns true if they match.
func ValidateKey(provided, expected string) bool {
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// KeyFromEnv reads the API key from the environment variable.
// Returns empty string if not set.
func KeyFromEnv() string {
	return os.Getenv(DefaultEnvVar)
}
