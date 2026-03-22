package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/szaher/agentspec/internal/ir"
	"github.com/szaher/agentspec/internal/k8s/converter"
	"github.com/szaher/agentspec/internal/parser"
)

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate resources from IntentLang files",
	}
	cmd.AddCommand(newGenerateCRDsCmd())
	return cmd
}

func newGenerateCRDsCmd() *cobra.Command {
	var (
		outputDir string
		namespace string
	)

	cmd := &cobra.Command{
		Use:   "crds [file.ias]",
		Short: "Generate Kubernetes CRD manifests from an IntentLang file",
		Long:  "Parse an IntentLang file and generate corresponding Kubernetes custom resource manifests.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]

			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("reading input file: %w", err)
			}

			// Parse IntentLang file to AST.
			astFile, parseErrors := parser.Parse(string(data), inputFile)
			if len(parseErrors) > 0 {
				for _, pe := range parseErrors {
					fmt.Fprintf(cmd.ErrOrStderr(), "parse error: %s\n", pe)
				}
				return fmt.Errorf("failed to parse %s: %d errors", inputFile, len(parseErrors))
			}

			// Lower AST to IR.
			doc, err := ir.Lower(astFile)
			if err != nil {
				return fmt.Errorf("lowering to IR: %w", err)
			}

			// Convert IR to Kubernetes CRD resources.
			resources, err := converter.ConvertDocument(doc, namespace)
			if err != nil {
				return fmt.Errorf("converting to CRDs: %w", err)
			}

			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}

			for _, res := range resources {
				yamlData, err := yaml.Marshal(res.Raw)
				if err != nil {
					return fmt.Errorf("marshaling %s/%s to YAML: %w", res.Kind, res.Name, err)
				}

				filename := fmt.Sprintf("%s_%s.yaml", res.Kind, res.Name)
				outPath := filepath.Join(outputDir, filename)
				if err := os.WriteFile(outPath, yamlData, 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", outPath, err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Generated %s\n", outPath)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated %d resources in %s\n", len(resources), outputDir)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "generated-crds", "Directory to write generated manifests")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace for generated resources")

	return cmd
}
