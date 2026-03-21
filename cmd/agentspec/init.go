package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/templates"
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

Run 'agentspec init' for an interactive template selector, or use
'agentspec init --template <name>' for non-interactive selection.
Use 'agentspec init --list-templates' to see all available templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listFlag {
				return listTemplates()
			}

			// Interactive selector when no --template flag
			if templateName == "" {
				selected, err := interactiveSelect(cmd)
				if err != nil {
					return err
				}
				templateName = selected
			}

			tmpl := templates.Get(templateName)
			if tmpl == nil {
				return fmt.Errorf("unknown template %q (use --list-templates to see available templates)", templateName)
			}

			if packageName == "" {
				packageName = templateName
			}

			if outputDir == "" {
				outputDir = "."
			}

			targetDir := filepath.Join(outputDir, packageName)

			// Check for existing files and prompt for overwrite
			if overwriteNeeded, err := checkExistingFiles(targetDir, tmpl, packageName); err != nil {
				return err
			} else if overwriteNeeded {
				if !confirmOverwrite(cmd, targetDir) {
					return fmt.Errorf("aborted")
				}
			}

			created, err := templates.ScaffoldDir(tmpl, targetDir, packageName)
			if err != nil {
				return fmt.Errorf("scaffold project: %w", err)
			}

			printSuccess(targetDir, packageName, tmpl, created)
			return nil
		},
	}

	cmd.Flags().StringVar(&templateName, "template", "", "Template name to use")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: current directory)")
	cmd.Flags().StringVar(&packageName, "name", "", "Package name (default: template name)")
	cmd.Flags().BoolVar(&listFlag, "list-templates", false, "List available templates")

	return cmd
}

func listTemplates() error {
	fmt.Println("Available templates:")
	fmt.Println()

	categories := []struct {
		name  string
		label string
	}{
		{"beginner", "Beginner"},
		{"intermediate", "Intermediate"},
		{"advanced", "Advanced"},
	}

	for _, cat := range categories {
		fmt.Printf("  %s:\n", cat.label)
		for _, t := range templates.All() {
			if t.Category == cat.name {
				fmt.Printf("    %-25s %-45s [%s]\n", t.Name, t.Description, t.Category)
			}
		}
		fmt.Println()
	}

	return nil
}

func interactiveSelect(cmd *cobra.Command) (string, error) {
	all := templates.All()

	fmt.Println("Choose a starter template:")
	fmt.Println()

	categories := []struct {
		name  string
		label string
	}{
		{"beginner", "Beginner"},
		{"intermediate", "Intermediate"},
		{"advanced", "Advanced"},
	}

	idx := 1
	indexMap := make(map[int]string)
	for _, cat := range categories {
		fmt.Printf("  %s:\n", cat.label)
		for _, t := range all {
			if t.Category == cat.name {
				fmt.Printf("    %d. %-22s %s\n", idx, t.Name, t.Description)
				indexMap[idx] = t.Name
				idx++
			}
		}
		fmt.Println()
	}

	fmt.Printf("Select template [1-%d]: ", idx-1)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input received")
	}

	input := strings.TrimSpace(scanner.Text())
	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num >= idx {
		return "", fmt.Errorf("invalid selection %q — enter a number between 1 and %d", input, idx-1)
	}

	return indexMap[num], nil
}

func checkExistingFiles(targetDir string, tmpl *templates.Template, packageName string) (bool, error) {
	iasFile := filepath.Join(targetDir, packageName+".ias")
	if _, err := os.Stat(iasFile); err == nil {
		return true, nil
	}
	if _, err := os.Stat(targetDir); err == nil {
		// Check if directory has any files
		entries, readErr := os.ReadDir(targetDir)
		if readErr == nil && len(entries) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func confirmOverwrite(cmd *cobra.Command, targetDir string) bool {
	fmt.Printf("Warning: %s already exists.\nOverwrite? [y/N]: ", targetDir)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	input := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return input == "y" || input == "yes"
}

func printSuccess(targetDir, packageName string, tmpl *templates.Template, created []string) {
	iasFile := filepath.Join(targetDir, packageName+".ias")

	fmt.Printf("Created project in %s/\n", targetDir)
	fmt.Println()
	fmt.Println("Files:")
	for _, f := range created {
		desc := ""
		if strings.HasSuffix(f, ".ias") {
			desc = "Agent definition"
		} else if strings.HasSuffix(f, "README.md") {
			desc = "Setup and run instructions"
		}
		fmt.Printf("  %-40s %s\n", filepath.Join(targetDir, f), desc)
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Set required environment variables:")
	fmt.Println("     export ANTHROPIC_API_KEY=\"your-key-here\"")
	fmt.Printf("  2. Validate: agentspec validate %s\n", iasFile)
	fmt.Printf("  3. Run:      agentspec run %s\n", iasFile)
}
