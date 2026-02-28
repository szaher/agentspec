package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/compiler"
)

func newCompileCmd() *cobra.Command {
	var (
		target        string
		outputDir     string
		platform      string
		name          string
		embedFrontend bool
		jsonOutput    bool
	)

	cmd := &cobra.Command{
		Use:   "compile [file.ias | directory]",
		Short: "Compile .ias files into a deployable agent artifact",
		Long: `Compile IntentLang (.ias) agent definitions into a standalone
executable or framework-specific source code.

The compiled binary is a self-contained agent service with health checks,
API endpoints, and optional built-in frontend.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve input files
			files, err := resolveCompileInputs(args)
			if err != nil {
				return err
			}

			if verbose {
				fmt.Fprintf(os.Stderr, "Compiling %d file(s)...\n", len(files))
			}

			opts := compiler.CompileOptions{
				Target:        target,
				OutputDir:     outputDir,
				Platform:      platform,
				Name:          name,
				EmbedFrontend: embedFrontend,
				Verbose:       verbose,
				JSONOutput:    jsonOutput,
			}

			result, err := compiler.Compile(files, opts)
			if err != nil {
				if jsonOutput {
					errJSON, _ := json.MarshalIndent(map[string]interface{}{
						"status":  "error",
						"message": err.Error(),
					}, "", "  ")
					fmt.Println(string(errJSON))
				}
				return fmt.Errorf("compilation failed: %w", err)
			}

			if jsonOutput {
				out, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(out))
			} else {
				printCompileResult(result, files)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "standalone", "Compilation target: standalone, crewai, langgraph, llamastack, llamaindex")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./build", "Output directory for compiled artifacts")
	cmd.Flags().StringVar(&platform, "platform", "", "Target platform (e.g., linux/amd64, darwin/arm64)")
	cmd.Flags().StringVar(&name, "name", "", "Output binary/project name (default: package name)")
	cmd.Flags().BoolVar(&embedFrontend, "embed-frontend", true, "Embed the built-in frontend in the compiled binary")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON result")

	return cmd
}

func resolveCompileInputs(args []string) ([]string, error) {
	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, fmt.Errorf("cannot access %q: %w", arg, err)
		}

		if info.IsDir() {
			// Find all .ias files in directory
			entries, err := os.ReadDir(arg)
			if err != nil {
				return nil, fmt.Errorf("reading directory %q: %w", arg, err)
			}
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".ias" {
					files = append(files, filepath.Join(arg, e.Name()))
				}
			}
		} else {
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .ias files found in the specified paths")
	}
	return files, nil
}

func printCompileResult(result *compiler.CompileResult, inputFiles []string) {
	// Header
	fmt.Printf("Compiling %s...\n", filepath.Base(inputFiles[0]))

	fmt.Printf("  ✓ Parsed %d file(s)\n", result.FilesProcessed)

	// Count agents
	fmt.Printf("  ✓ Compiled to %s binary\n", result.Target)
	fmt.Println()

	// Output info
	sizeMB := float64(result.SizeBytes) / (1024 * 1024)
	fmt.Printf("Output: %s (%.1f MB)\n", result.OutputPath, sizeMB)
	fmt.Printf("Platform: %s\n", result.Platform)
	fmt.Printf("Agents: %s\n", joinStrings(result.Agents))
	fmt.Printf("Config: %s\n", result.ConfigRef)
	fmt.Printf("Time: %dms\n", result.CompileTimeMS)

	if len(result.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return "(none)"
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += ", " + s
	}
	return result
}
