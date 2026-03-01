package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/apply"
	"github.com/szaher/designs/agentz/internal/events"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/policy"
	"github.com/szaher/designs/agentz/internal/state"

	// Register adapters
	_ "github.com/szaher/designs/agentz/internal/adapters/compose"
	_ "github.com/szaher/designs/agentz/internal/adapters/docker"
	_ "github.com/szaher/designs/agentz/internal/adapters/kubernetes"
	_ "github.com/szaher/designs/agentz/internal/adapters/local"
	_ "github.com/szaher/designs/agentz/internal/adapters/process"
)

func newApplyCmd() *cobra.Command {
	var (
		target      string
		env         string
		autoApprove bool
		planFile    string
		policyMode  string
		lockTimeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply desired state idempotently",
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

			backend := state.NewLocalBackend(stateFile).WithLockConfig(state.LockConfig{
				LockTimeout: lockTimeout,
			})
			current, err := backend.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			// Evaluate policy rules
			if len(doc.Policies) > 0 {
				mode := policy.ModeEnforce
				if policyMode == "warn" {
					mode = policy.ModeWarn
				}

				engine := policy.NewDefaultEngine()
				violations := engine.Evaluate(doc.Policies, doc.Resources)
				if len(violations) > 0 {
					output := policy.FormatViolations(violations, mode)
					if mode == policy.ModeEnforce {
						return fmt.Errorf("policy violations found:%s", output)
					}
					fmt.Fprintf(os.Stderr, "Policy warnings:%s\n", output)
				}
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

			applyCtx, cancel := context.WithTimeout(cmd.Context(), lockTimeout+5*time.Minute)
			defer cancel()

			emitter := &events.CollectorEmitter{}
			result, err := apply.Apply(
				applyCtx,
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
	cmd.Flags().StringVar(&policyMode, "policy", "enforce", "Policy evaluation mode: enforce (block on violations) or warn (report and proceed)")
	cmd.Flags().DurationVar(&lockTimeout, "lock-timeout", 30*time.Second, "Timeout for acquiring state file lock")

	_ = env      // will be used in Phase 6
	_ = planFile // will be used for plan-file support

	return cmd
}
