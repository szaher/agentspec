package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/plan"
)

func newDestroyCmd() *cobra.Command {
	var (
		target      string
		autoApprove bool
	)

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Tear down deployed resources",
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

			if !autoApprove {
				fmt.Printf("This will destroy all resources deployed via %q adapter.\n", adapterName)
				fmt.Print("Are you sure? (yes/no): ")
				var response string
				_, _ = fmt.Scanln(&response)
				if response != "yes" {
					fmt.Println("Destroy cancelled.")
					return nil
				}
			}

			results, err := adapter.Destroy(context.Background())
			if err != nil {
				return fmt.Errorf("destroy: %w", err)
			}

			succeeded := 0
			failed := 0
			for _, r := range results {
				if r.Status == adapters.ResultSuccess {
					succeeded++
				} else {
					failed++
					fmt.Fprintf(os.Stderr, "Failed to destroy %s: %s\n", r.FQN, r.Error)
				}
			}

			fmt.Printf("\n%d destroyed, %d failed\n", succeeded, failed)

			if failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Deploy target name")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")

	return cmd
}
