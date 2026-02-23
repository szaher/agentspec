package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/cli"
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

			exportDir := outDir
			if exportDir == "" {
				exportDir = "./export/"
			}

			if err := adapter.Export(context.Background(), doc.Resources, exportDir); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			fmt.Printf("Exported to %s (adapter: %s)\n", exportDir, binding.Adapter)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Binding name")
	cmd.Flags().StringVar(&env, "env", "", "Environment name")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Output directory (default: ./export/)")

	_ = env // will be used in Phase 6

	return cmd
}
