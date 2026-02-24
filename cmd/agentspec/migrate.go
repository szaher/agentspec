package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/szaher/designs/agentz/internal/formatter"
	"github.com/szaher/designs/agentz/internal/migrate"
	"github.com/szaher/designs/agentz/internal/parser"
)

func newMigrateCmd() *cobra.Command {
	var toV2 bool

	cmd := &cobra.Command{
		Use:   "migrate [path]",
		Short: "Migrate .az files to .ias or IntentLang 1.0 to 2.0",
		Long: `Migrate performs file and language version migrations:

Without flags: renames all .az files to .ias in the given directory.
With --to-v2: rewrites IntentLang 1.0 files to 2.0 syntax.
  - Replaces 'execution command "..."' with 'tool command { binary "..." }'
  - Replaces 'binding "name" adapter "..."' with 'deploy "name" target "..."'
  - Sets lang "2.0" in the package header`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if toV2 {
				files, err := resolveFiles(args)
				if err != nil {
					return err
				}
				return migrateToV2(files)
			}

			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			renamed, skipped, err := migrateAZToIAS(dir)
			if err != nil {
				return err
			}

			if renamed > 0 {
				fmt.Printf("\nMigration complete: %d file(s) renamed from .az to .ias.\n", renamed)
			} else if skipped == 0 {
				fmt.Println("No .az files found to migrate.")
			}
			if skipped > 0 {
				fmt.Fprintf(os.Stderr, "Warning: %d file(s) skipped due to conflicts (both .az and .ias exist).\n", skipped)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&toV2, "to-v2", false, "Migrate IntentLang 1.0 files to 2.0 syntax")

	return cmd
}

// migrateToV2 rewrites .ias files from IntentLang 1.0 to 2.0.
func migrateToV2(files []string) error {
	migrated := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		f, errs := parser.Parse(string(content), file)
		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: %s has parse errors, skipping\n", file)
			continue
		}

		if f.Package != nil && f.Package.LangVersion == "2.0" {
			fmt.Printf("  %s: already 2.0, skipping\n", file)
			continue
		}

		// Apply migration
		f = migrate.ToV2(f)

		// Format and write back
		output := formatter.Format(f)
		if err := os.WriteFile(file, []byte(output), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", file, err)
		}

		fmt.Printf("  Migrated: %s → IntentLang 2.0\n", filepath.Base(file))
		migrated++
	}

	if migrated > 0 {
		fmt.Printf("\n%d file(s) migrated to IntentLang 2.0.\n", migrated)
	} else {
		fmt.Println("No files needed migration to 2.0.")
	}

	return nil
}

// migrateAZToIAS walks the directory tree and renames .az files to .ias.
func migrateAZToIAS(root string) (renamed, skipped int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".az" {
			return nil
		}

		base := strings.TrimSuffix(path, ".az")
		iasPath := base + ".ias"

		if _, statErr := os.Stat(iasPath); statErr == nil {
			fmt.Fprintf(os.Stderr, "Conflict: both '%s' and '%s' exist — skipping.\n",
				filepath.Base(path), filepath.Base(iasPath))
			skipped++
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", path, readErr)
		}

		updated := strings.ReplaceAll(string(content), ".az", ".ias")
		if updated != string(content) {
			if writeErr := os.WriteFile(path, []byte(updated), info.Mode()); writeErr != nil {
				return fmt.Errorf("updating references in %s: %w", path, writeErr)
			}
		}

		if renameErr := os.Rename(path, iasPath); renameErr != nil {
			return fmt.Errorf("renaming %s: %w", path, renameErr)
		}

		fmt.Printf("  Renamed: %s → %s\n", filepath.Base(path), filepath.Base(iasPath))
		renamed++
		return nil
	})

	return renamed, skipped, err
}
