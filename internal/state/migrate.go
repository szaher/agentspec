package state

import (
	"fmt"
	"time"
)

// MigrationResult holds the outcome of a state migration.
type MigrationResult struct {
	Source   string
	Dest     string
	Migrated int
	Failed   int
	Skipped  int
	Errors   []string
	Duration time.Duration
}

// Migrate copies all state entries from src to dst.
// If dryRun is true, it reports what would be migrated without writing.
func Migrate(src, dst Backend, dryRun bool) (*MigrationResult, error) {
	start := time.Now()

	result := &MigrationResult{
		Source: fmt.Sprintf("%T", src),
		Dest:   fmt.Sprintf("%T", dst),
	}

	entries, err := src.Load()
	if err != nil {
		result.Duration = time.Since(start)
		return result, fmt.Errorf("source load failed: %w", err)
	}

	if entries == nil {
		// Source is empty (e.g. fresh backend with no state file).
		result.Duration = time.Since(start)
		return result, nil
	}

	if dryRun {
		result.Migrated = len(entries)
		result.Duration = time.Since(start)
		return result, nil
	}

	// Attempt bulk save first.
	if err := dst.Save(entries); err != nil {
		// Bulk save failed — try entry-by-entry to identify individual failures.
		for _, e := range entries {
			if saveErr := dst.Save([]Entry{e}); saveErr != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", e.FQN, saveErr))
			} else {
				result.Migrated++
			}
		}
		result.Duration = time.Since(start)
		if result.Failed > 0 {
			return result, fmt.Errorf("migration partially failed: %d of %d entries failed", result.Failed, len(entries))
		}
		return result, nil
	}

	result.Migrated = len(entries)
	result.Duration = time.Since(start)
	return result, nil
}
