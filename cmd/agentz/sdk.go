package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/sdk/generator"
)

func newSDKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sdk",
		Short: "SDK generation commands",
	}
	cmd.AddCommand(newSDKGenerateCmd())
	return cmd
}

func newSDKGenerateCmd() *cobra.Command {
	var (
		lang   string
		outDir string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate SDK for a target language",
		RunE: func(cmd *cobra.Command, args []string) error {
			language := generator.Language(lang)
			switch language {
			case generator.LangPython, generator.LangTypeScript, generator.LangGo:
			default:
				return fmt.Errorf("unsupported language: %s (use python, typescript, or go)", lang)
			}

			if outDir == "" {
				outDir = fmt.Sprintf("./sdk/%s/", lang)
			}

			cfg := generator.Config{
				Language: language,
				OutDir:   outDir,
			}

			if err := generator.Generate(cfg); err != nil {
				return fmt.Errorf("SDK generation failed: %w", err)
			}

			fmt.Printf("SDK generated at %s (language: %s)\n", outDir, lang)
			return nil
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "Target language (python|typescript|go)")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Output directory")
	_ = cmd.MarkFlagRequired("lang")

	return cmd
}
