package integration_tests

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestAllCLICommandsDocumented verifies that every CLI subcommand registered
// in newRootCmd() appears in the README.md CLI commands table.
// This prevents undocumented commands from shipping.
func TestAllCLICommandsDocumented(t *testing.T) {
	// All commands registered in cmd/agentspec/main.go newRootCmd()
	registeredCommands := []string{
		"version",
		"fmt",
		"validate",
		"plan",
		"apply",
		"diff",
		"export",
		"sdk",
		"migrate",
		"run",
		"dev",
		"status",
		"logs",
		"destroy",
		"init",
		"compile",
		"package",
		"eval",
		"publish",
		"install",
	}

	readmePath := "../README.md"
	f, err := os.Open(readmePath)
	if err != nil {
		t.Fatalf("failed to open README.md: %v", err)
	}
	defer f.Close()

	// Extract command names from the CLI commands table
	var readmeContent strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		readmeContent.WriteString(scanner.Text())
		readmeContent.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := readmeContent.String()

	for _, cmd := range registeredCommands {
		// Check that the command appears in a markdown table row: | `cmd` or | `cmd <
		if !strings.Contains(content, "| `"+cmd+"`") && !strings.Contains(content, "| `"+cmd+" ") {
			t.Errorf("command %q is registered in CLI but not documented in README.md", cmd)
		}
	}
}
