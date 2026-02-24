package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/templates"
)

func newInitCmd() *cobra.Command {
	var (
		templateName string
		outputDir    string
		packageName  string
		listFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new AgentSpec project from a template",
		Long: `Initialize a new AgentSpec project by scaffolding files from a built-in template.

Available templates:
  customer-support       Customer support agent with order lookup and KB tools
  rag-chatbot            RAG chatbot with document retrieval and semantic search
  code-review-pipeline   Multi-agent pipeline for automated code review
  data-extraction        Data extraction agent with structured output parsing
  research-assistant     Research assistant with web search and summarization`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listFlag {
				fmt.Println("Available templates:")
				fmt.Println()
				for _, t := range templates.All() {
					fmt.Printf("  %-25s %s\n", t.Name, t.Description)
				}
				return nil
			}

			if templateName == "" {
				return fmt.Errorf("--template flag is required (use --list to see available templates)")
			}

			tmpl := templates.Get(templateName)
			if tmpl == nil {
				return fmt.Errorf("unknown template %q (use --list to see available templates)", templateName)
			}

			content, err := templates.Content(tmpl)
			if err != nil {
				return fmt.Errorf("read template: %w", err)
			}

			if packageName == "" {
				packageName = templateName
			}

			// Replace template variables
			rendered := strings.ReplaceAll(string(content), "{{.PackageName}}", packageName)

			if outputDir == "" {
				outputDir = "."
			}

			outFile := filepath.Join(outputDir, packageName+".ias")

			// Check if file already exists
			if _, err := os.Stat(outFile); err == nil {
				return fmt.Errorf("file %q already exists (use a different --output-dir or --name)", outFile)
			}

			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}

			if err := os.WriteFile(outFile, []byte(rendered), 0644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}

			fmt.Printf("Created %s from template %q\n", outFile, templateName)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  1. Edit %s to customize your agent configuration\n", outFile)
			fmt.Printf("  2. Run: agentspec validate %s\n", outFile)
			fmt.Printf("  3. Run: agentspec apply %s\n", outFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&templateName, "template", "", "Template name to use")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: current directory)")
	cmd.Flags().StringVar(&packageName, "name", "", "Package name (default: template name)")
	cmd.Flags().BoolVar(&listFlag, "list", false, "List available templates")

	return cmd
}
