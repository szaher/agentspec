package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/plan"
)

func newExportCmd() *cobra.Command {
	var (
		target string
		env    string
		outDir string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export adapter-specific artifacts without applying",
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

			exportDir := outDir
			if exportDir == "" {
				exportDir = "./export/"
			}

			if err := adapter.Export(context.Background(), doc.Resources, exportDir); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			fmt.Printf("Exported to %s (adapter: %s)\n", exportDir, adapterName)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Binding name")
	cmd.Flags().StringVar(&env, "env", "", "Environment name")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Output directory (default: ./export/)")

	_ = env // will be used in Phase 6

	return cmd
}
