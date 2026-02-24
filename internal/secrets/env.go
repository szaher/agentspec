package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// EnvResolver resolves secret references of the form "env(VAR_NAME)"
// by reading from environment variables.
type EnvResolver struct{}

// NewEnvResolver creates an environment variable secret resolver.
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// Resolve looks up an env() reference and returns the value.
func (r *EnvResolver) Resolve(_ context.Context, ref string) (string, error) {
	if !strings.HasPrefix(ref, "env(") || !strings.HasSuffix(ref, ")") {
		return "", fmt.Errorf("unsupported secret reference format: %q (expected env(VAR_NAME))", ref)
	}

	varName := ref[4 : len(ref)-1]
	value, ok := os.LookupEnv(varName)
	if !ok {
		return "", fmt.Errorf("environment variable %q not set", varName)
	}

	return value, nil
}
