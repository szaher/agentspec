package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/plan"
)

func newLogsCmd() *cobra.Command {
	var (
		target string
		follow bool
		tail   int
		since  string
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream logs from deployed resources",
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

			opts := adapters.LogOptions{
				Follow: follow,
				Tail:   tail,
				Since:  since,
			}

			return adapter.Logs(context.Background(), os.Stdout, opts)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Deploy target name")
	cmd.Flags().BoolVar(&follow, "follow", false, "Follow log output")
	cmd.Flags().IntVar(&tail, "tail", 0, "Number of lines to show from end of logs")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since relative time (e.g. 5m, 1h)")

	return cmd
}
