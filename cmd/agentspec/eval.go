package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/evaluation"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/runtime"
)

func newEvalCmd() *cobra.Command {
	var (
		agentName  string
		tags       string
		output     string
		format     string
		compareTo  string
	)

	cmd := &cobra.Command{
		Use:   "eval [file.ias]",
		Short: "Run evaluation test cases against agents",
		Long: `Run declared evaluation test cases against agents defined in .ias files.
This validates agent quality by comparing actual outputs against expected outputs
using configurable scoring methods.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse and lower the .ias file
			files, err := resolveCompileInputs(args)
			if err != nil {
				return err
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			// Convert to runtime config
			config, err := runtime.FromIR(doc)
			if err != nil {
				return fmt.Errorf("config conversion failed: %w", err)
			}

			// Parse tags filter
			var tagFilter []string
			if tags != "" {
				tagFilter = strings.Split(tags, ",")
			}

			// Create a stub invoker for eval (in production, this would connect to a running agent)
			invoker := &stubInvoker{}
			runner := evaluation.NewRunner(invoker)

			// Run evals for matching agents
			var results []*evaluation.RunResult
			for _, agent := range config.Agents {
				if agentName != "" && agent.Name != agentName {
					continue
				}

				if len(agent.EvalCases) == 0 {
					if verbose {
						fmt.Fprintf(os.Stderr, "Agent %q has no eval cases, skipping\n", agent.Name)
					}
					continue
				}

				result, err := runner.Run(cmd.Context(), agent.Name, agent.EvalCases, tagFilter)
				if err != nil {
					return fmt.Errorf("eval failed for agent %q: %w", agent.Name, err)
				}
				results = append(results, result)
			}

			if len(results) == 0 {
				fmt.Println("No eval cases found for the specified agents.")
				return nil
			}

			// Format and output results
			for _, result := range results {
				report, err := evaluation.FormatReport(result, format)
				if err != nil {
					return fmt.Errorf("formatting report: %w", err)
				}

				if output != "" {
					if err := os.WriteFile(output, []byte(report), 0644); err != nil {
						return fmt.Errorf("writing report: %w", err)
					}
					fmt.Printf("Report written to %s\n", output)
				} else {
					fmt.Print(report)
				}

				// Compare if previous report provided
				if compareTo != "" {
					prev, err := loadPreviousResult(compareTo)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not load comparison report: %v\n", err)
					} else {
						fmt.Print(evaluation.CompareResults(result, prev))
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&agentName, "agent", "", "Evaluate specific agent by name")
	cmd.Flags().StringVar(&tags, "tags", "", "Filter eval cases by tags (comma-separated)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write report to file")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, markdown")
	cmd.Flags().StringVar(&compareTo, "compare", "", "Path to previous eval report for comparison")

	return cmd
}

func loadPreviousResult(path string) (*evaluation.RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result evaluation.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// stubInvoker is a placeholder that returns an error.
// In production use, the eval command would start or connect to a running agent.
type stubInvoker struct{}

func (s *stubInvoker) Invoke(_ context.Context, agentName, input string) (string, error) {
	return "", fmt.Errorf("agent %q not running — start the agent first or compile and run it", agentName)
}

// Ensure parseAndLower is available (defined in plan.go)
// Ensure resolveCompileInputs is available (defined in compile.go)
// Ensure parseAndLower signature matches usage
var _ = (*ir.Document)(nil)
