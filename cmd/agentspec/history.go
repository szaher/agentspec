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
	var stateFilePath string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show agent version history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentName == "" {
				return fmt.Errorf("--agent flag is required")
			}

			backend := &state.LocalBackend{Path: stateFilePath}
			versions, err := backend.GetVersions(agentName)
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
	cmd.Flags().StringVar(&stateFilePath, "state", ".agentspec.state.json", "State file path")

	return cmd
}
