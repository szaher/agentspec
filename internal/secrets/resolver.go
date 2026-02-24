// Package secrets defines the secret resolution abstraction for the AgentSpec runtime.
package secrets

import (
	"context"
)

// Resolver resolves secret references to their values.
type Resolver interface {
	// Resolve looks up a secret reference and returns its value.
	// The ref format depends on the implementation (e.g., "env(VAR_NAME)").
	Resolve(ctx context.Context, ref string) (string, error)
}
