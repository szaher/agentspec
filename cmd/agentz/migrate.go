package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Detect version mismatches and emit migration guidance",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAZFiles(args)
			if err != nil {
				return err
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return err
			}

			if doc.LangVersion != langVersion {
				fmt.Printf("Language version mismatch detected:\n")
				fmt.Printf("  File declares: lang %q\n", doc.LangVersion)
				fmt.Printf("  CLI supports:  lang %q\n", langVersion)
				fmt.Printf("\nMigration hints:\n")
				fmt.Printf("  - Update the 'lang' value in your package declaration\n")
				fmt.Printf("  - Review the changelog for breaking changes\n")
				fmt.Printf("  - Run 'agentz validate' after updating\n")
				return nil
			}

			fmt.Printf("Language version %q is up to date.\n", doc.LangVersion)
			return nil
		},
	}

	return cmd
}
