// Package main is the entry point for the agentspec CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set at build time.
var (
	version     = "0.3.0"
	langVersion = "3.0"
	irVersion   = "1.0"
)

// Global flags.
var (
	stateFile     string
	verbose       bool
	noColor       bool
	correlationID string
)

const (
	defaultStateFile = ".agentspec.state.json"
	oldStateFile     = ".agentz.state.json"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "agentspec",
		Short: "IntentLang agent packaging and deployment tool",
		Long: `AgentSpec parses IntentLang agent definitions (.ias files),
validates configurations, plans and applies changes idempotently
via pluggable adapters, and generates SDKs for multiple languages.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return migrateStateFile()
		},
	}

	root.PersistentFlags().StringVar(&stateFile, "state-file", defaultStateFile, "Path to state file")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	root.PersistentFlags().StringVar(&correlationID, "correlation-id", "", "Set explicit correlation ID")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newFmtCmd())
	root.AddCommand(newValidateCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newApplyCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newSDKCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newRunCmd())
	root.AddCommand(newDevCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newLogsCmd())
	root.AddCommand(newDestroyCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newCompileCmd())
	root.AddCommand(newPackageCmd())
	root.AddCommand(newEvalCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newInstallCmd())

	return root
}

// migrateStateFile auto-migrates old .agentz.state.json to .agentspec.state.json.
func migrateStateFile() error {
	if stateFile != defaultStateFile {
		return nil // user specified a custom path, don't migrate
	}

	if _, err := os.Stat(defaultStateFile); err == nil {
		return nil // new state file exists, nothing to migrate
	}

	if _, err := os.Stat(oldStateFile); err != nil {
		return nil // old state file doesn't exist either
	}

	// Auto-migrate
	if err := os.Rename(oldStateFile, defaultStateFile); err != nil {
		return fmt.Errorf("migrating state file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Notice: Migrated state file '%s' → '%s'\n", oldStateFile, defaultStateFile)
	return nil
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
