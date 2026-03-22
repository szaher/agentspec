package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/ir"
	"github.com/szaher/agentspec/internal/state"
)

func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage state backend",
	}

	cmd.AddCommand(newStateStatusCmd())
	cmd.AddCommand(newStateMigrateCmd())
	return cmd
}

// newStateStatusCmd creates the "agentspec state status" command.
func newStateStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [file.ias]",
		Short: "Show state backend health and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			backend, backendType, err := resolveBackendFromArgs(args)
			if err != nil {
				return err
			}
			if c, ok := backend.(state.Closer); ok {
				defer func() { _ = c.Close() }()
			}

			_, _ = fmt.Fprintf(os.Stdout, "State Backend: %s\n", backendType)

			// Health check
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()
			if hc, ok := backend.(state.HealthChecker); ok {
				if err := hc.Ping(ctx); err != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Status:        unreachable\n")
					_, _ = fmt.Fprintf(os.Stdout, "Error:         %v\n", err)
					os.Exit(2)
				}
			}
			_, _ = fmt.Fprintf(os.Stdout, "Status:        healthy\n")

			// Entry count and last write
			entries, err := backend.List(nil)
			if err != nil {
				return fmt.Errorf("listing entries: %w", err)
			}
			_, _ = fmt.Fprintf(os.Stdout, "Entries:       %d\n", len(entries))

			var latest time.Time
			for _, e := range entries {
				if e.LastApplied.After(latest) {
					latest = e.LastApplied
				}
			}
			if !latest.IsZero() {
				_, _ = fmt.Fprintf(os.Stdout, "Last Write:    %s\n", latest.Format(time.RFC3339))
			}

			return nil
		},
	}
	return cmd
}

// newStateMigrateCmd creates the "agentspec state migrate" command.
func newStateMigrateCmd() *cobra.Command {
	var (
		fromType string
		toType   string
		fromDSN  string
		toDSN    string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate state entries between backends",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromType == "" || toType == "" {
				return fmt.Errorf("both --from and --to are required")
			}

			srcProps := buildProps(fromType, fromDSN, "", "")
			src, err := state.New(fromType, srcProps)
			if err != nil {
				return fmt.Errorf("creating source backend: %w", err)
			}
			if c, ok := src.(state.Closer); ok {
				defer func() { _ = c.Close() }()
			}

			dstProps := buildProps(toType, toDSN, "", "")
			dst, err := state.New(toType, dstProps)
			if err != nil {
				return fmt.Errorf("creating destination backend: %w", err)
			}
			if c, ok := dst.(state.Closer); ok {
				defer func() { _ = c.Close() }()
			}

			result, err := state.Migrate(src, dst, dryRun)
			if err != nil {
				return err
			}

			if dryRun {
				_, _ = fmt.Fprintf(os.Stdout, "Dry run: %s → %s\n", fromType, toType)
				_, _ = fmt.Fprintf(os.Stdout, "Would migrate %d entries\n", result.Migrated)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Migrating state: %s → %s\n", fromType, toType)
				_, _ = fmt.Fprintf(os.Stdout, "\nMigration complete:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Migrated: %d\n", result.Migrated)
				_, _ = fmt.Fprintf(os.Stdout, "  Failed:   %d\n", result.Failed)
				_, _ = fmt.Fprintf(os.Stdout, "  Skipped:  %d\n", result.Skipped)
				_, _ = fmt.Fprintf(os.Stdout, "  Duration: %s\n", result.Duration.Round(100*time.Millisecond))
			}

			if result.Failed > 0 {
				os.Exit(3)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fromType, "from", "", "Source backend type (required)")
	cmd.Flags().StringVar(&toType, "to", "", "Destination backend type (required)")
	cmd.Flags().StringVar(&fromDSN, "from-dsn", "", "Source backend DSN")
	cmd.Flags().StringVar(&toDSN, "to-dsn", "", "Destination backend DSN")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be migrated without executing")

	return cmd
}

// resolveBackendFromArgs resolves the state backend from CLI flags, .ias file state block, or defaults.
func resolveBackendFromArgs(args []string) (state.Backend, string, error) {
	// CLI flag takes precedence
	if stateBackend != "" {
		props := buildProps(stateBackend, stateDSN, stateBucket, stateEndpoint)
		b, err := state.New(stateBackend, props)
		return b, stateBackend, err
	}

	// Try to resolve from .ias file state block
	if len(args) > 0 {
		files, err := resolveFiles(args)
		if err == nil {
			doc, err := parseAndLower(files)
			if err == nil && doc.StateConfig != nil {
				b, err := state.New(doc.StateConfig.Type, doc.StateConfig.Properties)
				return b, doc.StateConfig.Type, err
			}
		}
	}

	// Default to local
	b, err := state.New("local", map[string]string{"path": stateFile})
	return b, "local", err
}

// resolveStateBackend resolves the state backend from an IR document or CLI flags.
func resolveStateBackend(doc *ir.Document) (state.Backend, error) {
	// CLI flag takes precedence
	if stateBackend != "" {
		props := buildProps(stateBackend, stateDSN, stateBucket, stateEndpoint)
		return state.New(stateBackend, props)
	}

	// IR state config
	if doc != nil && doc.StateConfig != nil {
		return state.New(doc.StateConfig.Type, doc.StateConfig.Properties)
	}

	// Default: local backend
	return state.New("local", map[string]string{"path": stateFile})
}

// buildProps constructs backend-specific properties from CLI flags.
func buildProps(backendType, dsn, bucket, endpoint string) map[string]string {
	props := make(map[string]string)
	switch backendType {
	case "local":
		if stateFile != defaultStateFile {
			props["path"] = stateFile
		}
	case "postgres":
		if dsn != "" {
			props["dsn"] = dsn
		}
	case "etcd":
		if dsn != "" {
			props["endpoints"] = dsn
		}
		if endpoint != "" {
			props["endpoints"] = endpoint
		}
	case "s3":
		if bucket != "" {
			props["bucket"] = bucket
		}
		if endpoint != "" {
			props["endpoint"] = endpoint
		}
	case "kubernetes":
		// Uses in-cluster config by default
		if strings.Contains(dsn, "/") {
			parts := strings.SplitN(dsn, "/", 2)
			props["namespace"] = parts[0]
			props["name"] = parts[1]
		}
	}
	return props
}
