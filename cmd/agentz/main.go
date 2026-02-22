// Package main is the entry point for the agentz CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set at build time.
var (
	version     = "0.1.0"
	langVersion = "1.0"
	irVersion   = "1.0"
)

// Global flags.
var (
	stateFile     string
	verbose       bool
	noColor       bool
	correlationID string
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "agentz",
		Short: "Declarative agent packaging and deployment tool",
		Long: `Agentz parses declarative agent definitions (.az files),
validates configurations, plans and applies changes idempotently
via pluggable adapters, and generates SDKs for multiple languages.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&stateFile, "state-file", ".agentz.state.json", "Path to state file")
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

	return root
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
