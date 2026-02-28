package registry

import (
	"fmt"
	"log/slog"
)

// Client resolves packages using a fallback chain:
// local cache → lock file → Git resolution → checksum verification → cache storage.
type Client struct {
	cache    *Cache
	git      *GitResolver
	lockDeps map[string]LockedDep
	logger   *slog.Logger
}

// LockedDep represents a dependency pinned by the lock file.
type LockedDep struct {
	Source   string `json:"source"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// NewClient creates a new registry client.
func NewClient(cache *Cache, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		cache:    cache,
		git:      NewGitResolver(cache.Dir()),
		lockDeps: make(map[string]LockedDep),
		logger:   logger,
	}
}

// SetLockDeps sets the locked dependencies from a lock file.
func (c *Client) SetLockDeps(deps map[string]LockedDep) {
	c.lockDeps = deps
}

// Resolve resolves a package by source and version using the fallback chain.
func (c *Client) Resolve(source, version string) (*ResolvedPackage, error) {
	// 1. Check local cache
	if cachePath, ok := c.cache.Lookup(source, version); ok {
		c.logger.Debug("package found in cache", "source", source, "version", version)
		manifest, err := ReadManifest(cachePath)
		if err != nil {
			c.logger.Warn("cached manifest invalid, re-resolving", "source", source, "error", err)
		} else {
			checksum, _ := computeDirChecksum(cachePath)
			return &ResolvedPackage{
				Source:   source,
				Version:  version,
				Path:     cachePath,
				Manifest: manifest,
				Checksum: checksum,
			}, nil
		}
	}

	// 2. Check lock file for pinned version
	if locked, ok := c.lockDeps[source]; ok {
		if version == "" || version == locked.Version {
			c.logger.Debug("using locked version", "source", source, "version", locked.Version)
			version = locked.Version
		}
	}

	if version == "" {
		return nil, fmt.Errorf("package %q: no version specified and not in lock file", source)
	}

	// 3. Resolve from Git
	c.logger.Info("resolving package from Git", "source", source, "version", version)
	pkg, err := c.git.Resolve(source, version)
	if err != nil {
		return nil, fmt.Errorf("resolving %s@%s: %w", source, version, err)
	}

	// 4. Verify checksum against lock file if available
	if locked, ok := c.lockDeps[source]; ok && locked.Hash != "" {
		if pkg.Checksum != locked.Hash {
			return nil, fmt.Errorf("package %s@%s: checksum mismatch (expected %s, got %s)",
				source, version, locked.Hash, pkg.Checksum)
		}
		c.logger.Debug("checksum verified", "source", source, "version", version)
	}

	// 5. Emit info for unsigned packages
	if pkg.Manifest.Signature == "" {
		c.logger.Info("package is unsigned", "source", source, "version", version)
	}

	// 6. Store in cache
	if err := c.cache.Store(pkg); err != nil {
		c.logger.Warn("failed to cache package", "source", source, "error", err)
	}

	return pkg, nil
}

// ResolveDependencies resolves all transitive dependencies of a package.
func (c *Client) ResolveDependencies(manifest *Manifest) ([]*ResolvedPackage, error) {
	var resolved []*ResolvedPackage
	seen := make(map[string]bool)

	var resolve func(deps map[string]string) error
	resolve = func(deps map[string]string) error {
		for source, version := range deps {
			key := source + "@" + version
			if seen[key] {
				continue
			}
			seen[key] = true

			pkg, err := c.Resolve(source, version)
			if err != nil {
				return err
			}
			resolved = append(resolved, pkg)

			// Resolve transitive dependencies
			if pkg.Manifest.Dependencies != nil {
				if err := resolve(pkg.Manifest.Dependencies); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := resolve(manifest.Dependencies); err != nil {
		return nil, err
	}

	return resolved, nil
}
