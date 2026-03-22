// Package main is the entry point for the agentspec CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set at build time via ldflags.
var (
	version     = "dev"
	commit      = "unknown"
	date        = "unknown"
	langVersion = "3.0"
	irVersion   = "1.0"
)

// Global flags.
var (
	stateFile     string
	verbose       bool
	noColor       bool
	correlationID string

	// State backend flags (015-distributed-state-reconciliation)
	stateBackend  string
	stateDSN      string
	stateBucket   string
	stateEndpoint string
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
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("AgentSpec — declarative language for AI agents")
			fmt.Println()
			fmt.Println("Get started:")
			fmt.Println("  agentspec init              Create a new project from a template")
			fmt.Println("  agentspec --help             Show all available commands")
			fmt.Println()
			fmt.Println("Quickstart guide: https://szaher.github.io/agentspec/quickstart/")
			return nil
		},
	}

	root.PersistentFlags().StringVar(&stateFile, "state-file", defaultStateFile, "Path to state file")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	root.PersistentFlags().StringVar(&correlationID, "correlation-id", "", "Set explicit correlation ID")
	root.PersistentFlags().StringVar(&stateBackend, "state-backend", "", "Override state backend type (local, kubernetes, etcd, postgres, s3)")
	root.PersistentFlags().StringVar(&stateDSN, "state-dsn", "", "Backend DSN (postgres, etcd endpoints)")
	root.PersistentFlags().StringVar(&stateBucket, "state-bucket", "", "S3 bucket name")
	root.PersistentFlags().StringVar(&stateEndpoint, "state-endpoint", "", "Custom endpoint (S3-compatible, etcd)")

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
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newHistoryCmd())
	root.AddCommand(newOperatorCmd())
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newStateCmd())
	root.AddCommand(newGraphCmd())

	// Deprecation aliases for the run↔dev rename
	// Old 'run' (one-shot) behavior is now 'dev'
	root.AddCommand(newDeprecatedAlias("invoke", "dev", newDevCmd))
	// Old 'dev' (server) behavior is now 'run'
	root.AddCommand(newDeprecatedAlias("serve", "run", newRunCmd))

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

// newDeprecatedAlias creates a hidden command that delegates to another command
// while printing a deprecation warning to stderr.
func newDeprecatedAlias(oldName, newName string, buildCmd func() *cobra.Command) *cobra.Command {
	actual := buildCmd()
	alias := &cobra.Command{
		Use:    oldName,
		Short:  actual.Short,
		Long:   actual.Long,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Warning: '%s' is deprecated and will be removed in a future release. Use '%s' instead.\n", oldName, newName)
			actual.SetArgs(args)
			return actual.RunE(actual, args)
		},
	}
	// Copy flags from the actual command
	alias.Flags().AddFlagSet(actual.Flags())
	return alias
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
