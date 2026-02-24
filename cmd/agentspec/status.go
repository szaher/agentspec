package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/plan"
)

func newStatusCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show deployment status of resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveFiles(args)
			if err != nil {
				return err
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return err
			}

			adapterName := ""
			binding, _ := plan.ResolveBinding(doc.Bindings, target)
			if binding != nil {
				adapterName = binding.Adapter
			} else {
				dt, _ := plan.ResolveDeployTarget(doc.DeployTargets, target)
				if dt == nil {
					return fmt.Errorf("no deploy target found (use --target to specify)")
				}
				adapterName = plan.DeployTargetAdapter(dt.Target)
			}

			factory, err := adapters.Get(adapterName)
			if err != nil {
				return fmt.Errorf("adapter %q: %w", adapterName, err)
			}
			adapter := factory()

			statuses, err := adapter.Status(context.Background())
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			if len(statuses) == 0 {
				fmt.Println("No deployed resources found.")
				return nil
			}

			// Print status table
			fmt.Printf("%-40s %-12s %-10s %-10s %s\n", "NAME", "KIND", "STATE", "HEALTH", "ENDPOINT")
			fmt.Println(strings.Repeat("-", 90))
			for _, s := range statuses {
				endpoint := s.Endpoint
				if endpoint == "" {
					endpoint = "-"
				}
				health := s.Health
				if health == "" {
					health = "-"
				}
				fmt.Printf("%-40s %-12s %-10s %-10s %s\n",
					s.Name, s.Kind, s.State, health, endpoint)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Deploy target name")

	return cmd
}
