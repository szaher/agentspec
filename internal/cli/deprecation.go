package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckExtensionDeprecation checks if the given file uses the deprecated .az
// extension and emits a warning to stderr. It also detects conflicts where both
// .az and .ias versions of the same file exist in the same directory.
//
// Returns an error if a conflict is detected (both extensions exist).
// Returns nil and prints a deprecation warning if the file uses .az.
// Returns nil silently if the file uses .ias or another extension.
func CheckExtensionDeprecation(filePath string) error {
	ext := filepath.Ext(filePath)
	if ext != ".az" {
		return nil
	}

	// Check for conflict: does an .ias version also exist?
	base := strings.TrimSuffix(filePath, ".az")
	iasPath := base + ".ias"
	if _, err := os.Stat(iasPath); err == nil {
		return fmt.Errorf(
			"conflict: both '%s' and '%s' exist in the same directory; "+
				"remove one before proceeding, or run 'agentspec migrate' to resolve",
			filepath.Base(filePath), filepath.Base(iasPath),
		)
	}

	fmt.Fprintf(os.Stderr,
		"Warning: '%s' uses the deprecated '.az' extension. "+
			"Use '.ias' instead. Run 'agentspec migrate' to rename files.\n",
		filepath.Base(filePath),
	)
	return nil
}
