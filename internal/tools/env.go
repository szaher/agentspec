package tools

import (
	"fmt"
	"os"
	"os/user"
)

// SafeEnv returns a minimal environment for tool execution.
// Includes only PATH, HOME, and configured secrets.
// MUST NOT inherit the full host environment.
func SafeEnv(secrets map[string]string) []string {
	env := make([]string, 0, 2+len(secrets))

	// PATH: use system default or inherit from host
	pathVal := "/usr/local/bin:/usr/bin:/bin"
	if p := os.Getenv("PATH"); p != "" {
		pathVal = p
	}
	env = append(env, fmt.Sprintf("PATH=%s", pathVal))

	// HOME: use current user's home directory
	homeVal := os.Getenv("HOME")
	if homeVal == "" {
		if u, err := user.Current(); err == nil {
			homeVal = u.HomeDir
		}
	}
	if homeVal != "" {
		env = append(env, fmt.Sprintf("HOME=%s", homeVal))
	}

	// Add configured secrets
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}
