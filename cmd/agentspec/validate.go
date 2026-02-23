package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/cli"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/validate"
)

func newValidateCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "validate [files...]",
		Short: "Validate IntentLang definitions (structural + semantic)",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAZFiles(args)
			if err != nil {
				return err
			}

			var allErrors []*validate.ValidationError

			for _, file := range files {
				if err := cli.CheckExtensionDeprecation(file); err != nil {
					return err
				}

				input, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("cannot read %s: %w", file, err)
				}

				f, parseErrs := parser.Parse(string(input), file)
				if parseErrs != nil {
					for _, e := range parseErrs {
						allErrors = append(allErrors, &validate.ValidationError{
							File:    e.File,
							Line:    e.Line,
							Column:  e.Column,
							Message: e.Message,
							Hint:    e.Hint,
						})
					}
					continue
				}

				structErrs := validate.ValidateStructural(f)
				allErrors = append(allErrors, structErrs...)

				semErrs := validate.ValidateSemantic(f)
				allErrors = append(allErrors, semErrs...)
			}

			if len(allErrors) == 0 {
				return nil
			}

			switch format {
			case "json":
				type jsonError struct {
					File    string `json:"file"`
					Line    int    `json:"line"`
					Column  int    `json:"column"`
					Message string `json:"message"`
					Hint    string `json:"hint,omitempty"`
				}
				var out []jsonError
				for _, e := range allErrors {
					out = append(out, jsonError{
						File:    e.File,
						Line:    e.Line,
						Column:  e.Column,
						Message: e.Message,
						Hint:    e.Hint,
					})
				}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Fprintln(os.Stderr, string(data))
			default:
				for _, e := range allErrors {
					fmt.Fprintln(os.Stderr, e.Error())
				}
			}

			os.Exit(1)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format (text|json)")

	return cmd
}
