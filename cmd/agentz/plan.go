package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/state"
)

func newPlanCmd() *cobra.Command {
	var (
		target string
		env    string
		format string
		out    string
	)

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show what changes would be made without applying",
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

			binding, _ := plan.ResolveBinding(doc.Bindings, target)

			p := plan.ComputePlan(doc.Resources, current)
			if binding != nil {
				p.TargetBinding = binding.Adapter + " (binding " + fmt.Sprintf("%q", binding.Name) + ")"
			}

			var output string
			switch format {
			case "json":
				output, err = plan.FormatJSON(p)
				if err != nil {
					return err
				}
			default:
				output = plan.FormatText(p)
			}

			if out != "" {
				if err := os.WriteFile(out, []byte(output), 0644); err != nil {
					return err
				}
			} else {
				fmt.Print(output)
			}

			if p.HasChanges {
				os.Exit(2)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Binding name")
	cmd.Flags().StringVar(&env, "env", "", "Environment name")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text|json)")
	cmd.Flags().StringVar(&out, "out", "", "Write plan to file")

	_ = env // will be used in Phase 6

	return cmd
}

// parseAndLower parses all .az files and produces a single IR document.
func parseAndLower(files []string) (*ir.Document, error) {
	if len(files) == 0 {
		files, _ = resolveAZFiles(nil)
	}

	// For MVP, parse first file only (single-file packages)
	if len(files) == 0 {
		return nil, fmt.Errorf("no .az files found")
	}

	input, err := os.ReadFile(files[0])
	if err != nil {
		return nil, err
	}

	f, parseErrs := parser.Parse(string(input), files[0])
	if parseErrs != nil {
		for _, e := range parseErrs {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return nil, fmt.Errorf("parse errors")
	}

	return ir.Lower(f)
}
