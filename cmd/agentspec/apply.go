package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/apply"
	"github.com/szaher/designs/agentz/internal/cli"
	"github.com/szaher/designs/agentz/internal/events"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/state"

	// Register adapters
	_ "github.com/szaher/designs/agentz/internal/adapters/compose"
	_ "github.com/szaher/designs/agentz/internal/adapters/local"
)

func newApplyCmd() *cobra.Command {
	var (
		target      string
		env         string
		autoApprove bool
		planFile    string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply desired state idempotently",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAZFiles(args)
			if err != nil {
				return err
			}

			for _, file := range files {
				if err := cli.CheckExtensionDeprecation(file); err != nil {
					return err
				}
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return err
			}

			binding, _ := plan.ResolveBinding(doc.Bindings, target)
			if binding == nil {
				return fmt.Errorf("no binding found (use --target to specify)")
			}

			factory, err := adapters.Get(binding.Adapter)
			if err != nil {
				return fmt.Errorf("adapter %q: %w", binding.Adapter, err)
			}
			adapter := factory()

			backend := state.NewLocalBackend(stateFile)
			current, err := backend.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			p := plan.ComputePlan(doc.Resources, current)
			if !p.HasChanges {
				fmt.Println("No changes. Infrastructure is up-to-date.")
				return nil
			}

			if !autoApprove {
				fmt.Print(plan.FormatText(p))
				fmt.Print("\nDo you want to apply these changes? (yes/no): ")
				var response string
				_, _ = fmt.Scanln(&response)
				if response != "yes" {
					fmt.Println("Apply cancelled.")
					return nil
				}
			}

			cid := correlationID
			if cid == "" {
				cid = "apply-" + fmt.Sprintf("%d", os.Getpid())
			}

			emitter := &events.CollectorEmitter{}
			result, err := apply.Apply(
				context.Background(),
				adapter,
				p.Actions,
				backend,
				emitter,
				cid,
			)
			if err != nil {
				return err
			}

			fmt.Printf("\n%d created, %d updated, %d deleted, %d failed\n",
				result.Created, result.Updated, result.Deleted, result.Failed)
			fmt.Printf("State saved to %s\n", stateFile)

			if result.Failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Binding name")
	cmd.Flags().StringVar(&env, "env", "", "Environment name")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&planFile, "plan-file", "", "Use a saved plan file")

	_ = env      // will be used in Phase 6
	_ = planFile // will be used for plan-file support

	return cmd
}
