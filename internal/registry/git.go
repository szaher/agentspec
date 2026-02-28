package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitResolver resolves packages from Git repositories.
type GitResolver struct {
	cacheDir string
}

// NewGitResolver creates a Git-based package resolver.
func NewGitResolver(cacheDir string) *GitResolver {
	return &GitResolver{cacheDir: cacheDir}
}

// Resolve clones or fetches a package from a Git URL, checks out the version tag,
// reads the manifest, and verifies the checksum.
func (r *GitResolver) Resolve(source, version string) (*ResolvedPackage, error) {
	// Parse the source into host/path
	repoURL := "https://" + source + ".git"

	// Determine cache path
	cachePath := r.cachePath(source, version)

	// Check if already cached
	if _, err := os.Stat(filepath.Join(cachePath, ManifestFile)); err == nil {
		manifest, err := ReadManifest(cachePath)
		if err != nil {
			return nil, fmt.Errorf("reading cached manifest: %w", err)
		}
		checksum, err := computeDirChecksum(cachePath)
		if err != nil {
			return nil, fmt.Errorf("computing checksum: %w", err)
		}
		return &ResolvedPackage{
			Source:   source,
			Version:  version,
			Path:     cachePath,
			Manifest: manifest,
			Checksum: checksum,
		}, nil
	}

	// Clone/fetch the repository
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	// Clone with specific tag
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", tag, repoURL, cachePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone %s@%s: %s: %w", source, tag, string(out), err)
	}

	// Read and validate manifest
	manifest, err := ReadManifest(cachePath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest from %s: %w", source, err)
	}

	// Compute checksum
	checksum, err := computeDirChecksum(cachePath)
	if err != nil {
		return nil, fmt.Errorf("computing checksum: %w", err)
	}

	return &ResolvedPackage{
		Source:   source,
		Version:  version,
		Path:     cachePath,
		Manifest: manifest,
		Checksum: checksum,
	}, nil
}

// ListVersions lists available version tags from a Git repository.
func (r *GitResolver) ListVersions(source string) ([]string, error) {
	repoURL := "https://" + source + ".git"

	cmd := exec.Command("git", "ls-remote", "--tags", "--refs", repoURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote %s: %w", source, err)
	}

	var versions []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := parts[1]
		tag := strings.TrimPrefix(ref, "refs/tags/")
		tag = strings.TrimPrefix(tag, "v")
		if isValidSemver(tag) {
			versions = append(versions, tag)
		}
	}

	return versions, nil
}

// cachePath returns the local cache directory for a package version.
func (r *GitResolver) cachePath(source, version string) string {
	// Cache layout: ~/.agentspec/cache/<host>/<path>/@v/<version>/
	return filepath.Join(r.cacheDir, source, "@v", version)
}

// ResolvedPackage represents a package resolved from a Git repository.
type ResolvedPackage struct {
	Source   string
	Version  string
	Path     string
	Manifest *Manifest
	Checksum string
}

// computeDirChecksum computes a SHA-256 checksum of all .ias and .yaml files in a directory.
func computeDirChecksum(dir string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip .git directory
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ias" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(dir, path)
		h.Write([]byte(relPath))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
