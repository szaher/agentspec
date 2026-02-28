package compiler

import (
	"fmt"
	"runtime"
)

// SupportedPlatforms lists the target platforms for cross-compilation.
var SupportedPlatforms = []string{
	"linux/amd64",
	"linux/arm64",
	"darwin/amd64",
	"darwin/arm64",
	"windows/amd64",
}

// CurrentPlatform returns the current OS/arch string.
func CurrentPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// ValidatePlatform checks if the platform string is supported.
func ValidatePlatform(platform string) error {
	if platform == "" {
		return nil // will use current platform
	}
	for _, p := range SupportedPlatforms {
		if p == platform {
			return nil
		}
	}
	return fmt.Errorf("unsupported platform %q, supported: %v", platform, SupportedPlatforms)
}
