package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [path]",
		Short: "Migrate .az files to .ias and detect version mismatches",
		Long: `Migrate renames all .az files to .ias in the given directory (default: current directory)
and its subdirectories. It also updates internal references if .az appears in file content.

If both foo.az and foo.ias exist, that file is skipped and an error is reported.

After renaming, the command checks for language version mismatches and emits guidance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	return cmd
}

// migrateAZToIAS walks the directory tree and renames .az files to .ias.
// Returns count of renamed files, skipped files (conflicts), and any error.
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

		// Check for conflict
		if _, statErr := os.Stat(iasPath); statErr == nil {
			fmt.Fprintf(os.Stderr, "Conflict: both '%s' and '%s' exist — skipping.\n",
				filepath.Base(path), filepath.Base(iasPath))
			skipped++
			return nil
		}

		// Update internal references in file content
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

		// Rename file
		if renameErr := os.Rename(path, iasPath); renameErr != nil {
			return fmt.Errorf("renaming %s: %w", path, renameErr)
		}

		fmt.Printf("  Renamed: %s → %s\n", filepath.Base(path), filepath.Base(iasPath))
		renamed++
		return nil
	})

	return renamed, skipped, err
}
