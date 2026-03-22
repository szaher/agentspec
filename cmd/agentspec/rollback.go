package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/state"
)

func newRollbackCmd() *cobra.Command {
	var agentName string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback agent to previous version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentName == "" {
				return fmt.Errorf("--agent flag is required")
			}

			backend, _, err := resolveBackendFromArgs(args)
			if err != nil {
				return fmt.Errorf("resolving state backend: %w", err)
			}
			if c, ok := backend.(state.Closer); ok {
				defer func() { _ = c.Close() }()
			}

			vs, ok := backend.(state.VersionStore)
			if !ok {
				return fmt.Errorf("current state backend does not support version history")
			}

			versions, err := vs.GetVersions(agentName)
			if err != nil {
				return fmt.Errorf("load versions: %w", err)
			}

			if len(versions) < 2 {
				return fmt.Errorf("no previous version to rollback to for agent %q", agentName)
			}

			// Restore the second-to-last version
			prev := versions[len(versions)-2]

			// Create a rollback version entry
			rollbackEntry := state.VersionEntry{
				Version:   versions[len(versions)-1].Version + 1,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Summary:   fmt.Sprintf("Rollback to version %d", prev.Version),
				Snapshot:  prev.Snapshot,
			}

			if err := vs.SaveVersion(agentName, rollbackEntry); err != nil {
				return fmt.Errorf("save rollback version: %w", err)
			}

			fmt.Printf("Rolled back agent %q to version %d (now version %d)\n",
				agentName, prev.Version, rollbackEntry.Version)
			return nil
		},
	}

	cmd.Flags().StringVar(&agentName, "agent", "", "Agent name to rollback")

	return cmd
}
