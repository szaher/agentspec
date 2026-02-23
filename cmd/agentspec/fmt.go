package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/cli"
	"github.com/szaher/designs/agentz/internal/formatter"
	"github.com/szaher/designs/agentz/internal/parser"
)

func newFmtCmd() *cobra.Command {
	var (
		check bool
		diff  bool
	)

	cmd := &cobra.Command{
		Use:   "fmt [files...]",
		Short: "Format IntentLang source files to canonical style",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAZFiles(args)
			if err != nil {
				return err
			}

			anyChanged := false
			for _, file := range files {
				if err := cli.CheckExtensionDeprecation(file); err != nil {
					return err
				}

				changed, err := formatFile(file, check, diff)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", file, err)
					return err
				}
				if changed {
					anyChanged = true
				}
			}

			if check && anyChanged {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Report whether files need formatting without writing")
	cmd.Flags().BoolVar(&diff, "diff", false, "Print diff of changes to stdout")

	return cmd
}

func formatFile(path string, check, showDiff bool) (bool, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	f, parseErrs := parser.Parse(string(input), path)
	if parseErrs != nil {
		for _, e := range parseErrs {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return false, fmt.Errorf("parse errors in %s", path)
	}

	formatted := formatter.Format(f)

	if string(input) == formatted {
		return false, nil
	}

	if check {
		fmt.Printf("%s needs formatting\n", path)
		return true, nil
	}

	if showDiff {
		fmt.Printf("--- %s\n+++ %s (formatted)\n", path, path)
		// Simple line-level diff indication
		fmt.Println(formatted)
	}

	if !check {
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			return true, err
		}
	}

	return true, nil
}

func resolveAZFiles(args []string) ([]string, error) {
	if len(args) == 0 {
		args = []string{"."}
	}

	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", arg, err)
		}
		if info.IsDir() {
			// Prefer .ias files; also include .az for backward compatibility
			iasMatches, err := filepath.Glob(filepath.Join(arg, "*.ias"))
			if err != nil {
				return nil, err
			}
			files = append(files, iasMatches...)
			azMatches, err := filepath.Glob(filepath.Join(arg, "*.az"))
			if err != nil {
				return nil, err
			}
			files = append(files, azMatches...)
		} else {
			files = append(files, arg)
		}
	}
	return files, nil
}

func init() {
	// Will be added to root in main.go
}
