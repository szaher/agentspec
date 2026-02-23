package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/state"
)

func newDiffCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show drift between desired state and actual state",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAZFiles(args)
			if err != nil {
				return err
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return err
			}

			backend := state.NewLocalBackend(stateFile)
			current, err := backend.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			drift := plan.DetectDrift(doc.Resources, current)
			if !drift.HasDrift {
				fmt.Println("No drift detected.")
				return nil
			}

			fmt.Printf("Drift detected: %d resource(s)\n\n", len(drift.Drifted))
			for _, d := range drift.Drifted {
				switch d.Type {
				case "missing":
					fmt.Printf("  MISSING  %s (expected in state but not found)\n", d.FQN)
				case "hash_mismatch":
					fmt.Printf("  CHANGED  %s\n", d.FQN)
					fmt.Printf("           expected: %s\n", d.Expected)
					fmt.Printf("           actual:   %s\n", d.Actual)
				case "extra":
					fmt.Printf("  EXTRA    %s (in state but not in definitions)\n", d.FQN)
				}
			}

			os.Exit(2)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Binding name")

	return cmd
}
