package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultCacheDir returns the default package cache directory.
func DefaultCacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentspec", "cache")
}

// Cache manages the local package cache.
type Cache struct {
	dir string
}

// NewCache creates a new local package cache.
func NewCache(dir string) *Cache {
	if dir == "" {
		dir = DefaultCacheDir()
	}
	return &Cache{dir: dir}
}

// Dir returns the cache directory.
func (c *Cache) Dir() string {
	return c.dir
}

// Lookup checks if a package version exists in the cache and returns its path.
func (c *Cache) Lookup(source, version string) (string, bool) {
	path := c.path(source, version)
	if _, err := os.Stat(filepath.Join(path, ManifestFile)); err == nil {
		return path, true
	}
	return "", false
}

// Store saves a resolved package to the cache.
func (c *Cache) Store(pkg *ResolvedPackage) error {
	destPath := c.path(pkg.Source, pkg.Version)
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("creating cache path: %w", err)
	}

	// If the package is already at the expected cache path, nothing to do
	if pkg.Path == destPath {
		return nil
	}

	// Copy package files to cache
	return copyDir(pkg.Path, destPath)
}

// Invalidate removes a cached package version.
func (c *Cache) Invalidate(source, version string) error {
	path := c.path(source, version)
	return os.RemoveAll(path)
}

// List returns all cached packages.
func (c *Cache) List() ([]CachedPackage, error) {
	var packages []CachedPackage

	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		return packages, nil
	}

	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() != ManifestFile {
			return nil
		}

		manifest, err := ReadManifest(filepath.Dir(path))
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(c.dir, filepath.Dir(path))
		// Extract source and version from path: <source>/@v/<version>
		parts := strings.Split(relPath, "/@v/")
		if len(parts) != 2 {
			return nil
		}

		packages = append(packages, CachedPackage{
			Source:   parts[0],
			Version:  parts[1],
			Path:     filepath.Dir(path),
			Manifest: manifest,
		})
		return nil
	})

	return packages, err
}

func (c *Cache) path(source, version string) string {
	return filepath.Join(c.dir, source, "@v", version)
}

// CachedPackage represents a package stored in the local cache.
type CachedPackage struct {
	Source   string
	Version  string
	Path     string
	Manifest *Manifest
}

// copyDir copies all files from src to dst recursively.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}
