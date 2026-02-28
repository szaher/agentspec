package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/registry"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <package@version>",
		Short: "Install a package from a Git repository",
		Long:  "Resolves a package from its Git repository, downloads it to the local cache, and updates the lock file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(args[0])
		},
	}

	return cmd
}

func runInstall(packageRef string) error {
	// Parse package@version
	source, version := parsePackageRef(packageRef)
	if source == "" {
		return fmt.Errorf("invalid package reference %q (expected source@version)", packageRef)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cache := registry.NewCache("")
	client := registry.NewClient(cache, logger)

	// Load existing lock file
	lockDeps := loadLockDeps()
	client.SetLockDeps(lockDeps)

	fmt.Printf("Resolving %s@%s...\n", source, version)

	pkg, err := client.Resolve(source, version)
	if err != nil {
		return fmt.Errorf("resolving package: %w", err)
	}

	fmt.Printf("Installed %s@%s to %s\n", pkg.Source, pkg.Version, pkg.Path)

	// Resolve transitive dependencies
	deps, err := client.ResolveDependencies(pkg.Manifest)
	if err != nil {
		return fmt.Errorf("resolving dependencies: %w", err)
	}
	for _, dep := range deps {
		fmt.Printf("  dependency: %s@%s\n", dep.Source, dep.Version)
	}

	// Update lock file
	if err := updateLockFile(pkg, deps); err != nil {
		return fmt.Errorf("updating lock file: %w", err)
	}

	fmt.Printf("Lock file updated.\n")
	return nil
}

func parsePackageRef(ref string) (source, version string) {
	// Handle @version suffix
	if idx := lastIndex(ref, '@'); idx > 0 {
		source = ref[:idx]
		version = ref[idx+1:]
		return
	}
	// No version — use latest
	return ref, ""
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func loadLockDeps() map[string]registry.LockedDep {
	deps := make(map[string]registry.LockedDep)
	data, err := os.ReadFile(".agentspec.lock")
	if err != nil {
		return deps
	}
	var lockFile struct {
		Dependencies []struct {
			Source  string `json:"source"`
			Version string `json:"version"`
			Hash    string `json:"hash"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return deps
	}
	for _, d := range lockFile.Dependencies {
		deps[d.Source] = registry.LockedDep{
			Source:  d.Source,
			Version: d.Version,
			Hash:    d.Hash,
		}
	}
	return deps
}

func updateLockFile(pkg *registry.ResolvedPackage, deps []*registry.ResolvedPackage) error {
	type lockDep struct {
		Source  string `json:"source"`
		Version string `json:"version"`
		Hash    string `json:"hash"`
		Path    string `json:"resolved_path"`
	}

	// Load existing lock file
	var lockFile struct {
		Version      int       `json:"version"`
		Dependencies []lockDep `json:"dependencies"`
	}

	data, err := os.ReadFile(".agentspec.lock")
	if err == nil {
		_ = json.Unmarshal(data, &lockFile)
	}

	lockFile.Version = 1

	// Build index of existing deps
	existing := make(map[string]int)
	for i, d := range lockFile.Dependencies {
		existing[d.Source] = i
	}

	// Upsert the main package
	newDep := lockDep{
		Source:  pkg.Source,
		Version: pkg.Version,
		Hash:    pkg.Checksum,
		Path:    pkg.Path,
	}
	if idx, ok := existing[pkg.Source]; ok {
		lockFile.Dependencies[idx] = newDep
	} else {
		lockFile.Dependencies = append(lockFile.Dependencies, newDep)
	}

	// Upsert transitive dependencies
	for _, dep := range deps {
		d := lockDep{
			Source:  dep.Source,
			Version: dep.Version,
			Hash:    dep.Checksum,
			Path:    dep.Path,
		}
		if idx, ok := existing[dep.Source]; ok {
			lockFile.Dependencies[idx] = d
		} else {
			lockFile.Dependencies = append(lockFile.Dependencies, d)
		}
	}

	out, err := json.MarshalIndent(lockFile, "", "  ")
	if err != nil {
		return err
	}

	lockPath := filepath.Join(".", ".agentspec.lock")
	return os.WriteFile(lockPath, out, 0644)
}
