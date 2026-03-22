package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/state"
)

func newHistoryCmd() *cobra.Command {
	var agentName string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show agent version history",
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

			if len(versions) == 0 {
				fmt.Printf("No version history for agent %q\n", agentName)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "VERSION\tTIMESTAMP\tSUMMARY"); err != nil {
				return err
			}
			for _, v := range versions {
				if _, err := fmt.Fprintf(w, "%d\t%s\t%s\n", v.Version, v.Timestamp, v.Summary); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&agentName, "agent", "", "Agent name to show history for")

	return cmd
}
