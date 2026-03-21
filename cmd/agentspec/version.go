package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("agentspec %s (commit %s, built %s, lang %s, ir %s)\n", version, commit, date, langVersion, irVersion)
			checkLatestVersion()
		},
	}
}

func checkLatestVersion() {
	if version == "dev" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/szaher/agentspec/releases/latest", nil)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(version, "v")

	if latest != "" && latest != current && isNewer(latest, current) {
		fmt.Println()
		fmt.Printf("Update available: %s → %s\n", current, latest)
		fmt.Printf("  Release: %s\n", release.HTMLURL)
		fmt.Println("  brew upgrade agentspec                          (macOS)")
		fmt.Println("  docker pull ghcr.io/szaher/agentspec:latest")
	}
}

// isNewer returns true if a is newer than b using simple semver comparison.
func isNewer(a, b string) bool {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] > bParts[i] {
			return true
		}
		if aParts[i] < bParts[i] {
			return false
		}
	}
	return len(aParts) > len(bParts)
}
