package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/registry"
)

func newPublishCmd() *cobra.Command {
	var sign bool

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish an AgentPack package to a Git remote",
		Long:  "Reads agentpack.yaml, validates the package, creates a version tag, and pushes to the configured Git remote.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(sign)
		},
	}

	cmd.Flags().BoolVar(&sign, "sign", false, "Sign the package (not yet implemented)")

	return cmd
}

func runPublish(sign bool) error {
	if sign {
		fmt.Println("Package signing is not yet implemented. Publishing unsigned package.")
	}

	// Read and validate manifest
	manifest, err := registry.ReadManifest(".")
	if err != nil {
		return fmt.Errorf("reading manifest: %w\nMake sure agentpack.yaml exists in the current directory", err)
	}

	fmt.Printf("Publishing %s@%s\n", manifest.Name, manifest.Version)

	// Validate all exported files exist
	for _, export := range manifest.Exports {
		if _, err := os.Stat(export); os.IsNotExist(err) {
			return fmt.Errorf("exported file %q does not exist", export)
		}
	}

	// Check git status
	statusCmd := exec.CommandContext(context.Background(), "git", "status", "--porcelain")
	statusOut, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(statusOut))) > 0 {
		return fmt.Errorf("working directory has uncommitted changes; commit before publishing")
	}

	// Create version tag
	tag := "v" + manifest.Version
	tagCmd := exec.CommandContext(context.Background(), "git", "tag", "-a", tag, "-m", fmt.Sprintf("Release %s", manifest.FullName()))
	if out, err := tagCmd.CombinedOutput(); err != nil {
		// Tag might already exist
		if strings.Contains(string(out), "already exists") {
			return fmt.Errorf("tag %s already exists; bump version in agentpack.yaml", tag)
		}
		return fmt.Errorf("git tag: %s: %w", string(out), err)
	}

	// Push tag
	pushCmd := exec.CommandContext(context.Background(), "git", "push", "origin", tag)
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push tag: %s: %w", string(out), err)
	}

	fmt.Printf("Published %s (tag: %s)\n", manifest.FullName(), tag)
	return nil
}
