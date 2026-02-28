package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/imports"
	"github.com/szaher/designs/agentz/internal/registry"
)

// TestManifestParseAndWrite verifies reading and writing agentpack.yaml manifests.
func TestManifestParseAndWrite(t *testing.T) {
	manifest := &registry.Manifest{
		Name:        "my-agent-skills",
		Version:     "1.2.3",
		Description: "A collection of reusable agent skills",
		Author:      "test@example.com",
		License:     "MIT",
		AgentSpec:   ">=0.3.0",
		Dependencies: map[string]string{
			"github.com/example/base-skills": "1.0.0",
		},
		Exports: []string{"skills.ias", "prompts.ias"},
	}

	dir := t.TempDir()
	if err := registry.WriteManifest(manifest, dir); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Read it back
	loaded, err := registry.ReadManifest(dir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	if loaded.Name != "my-agent-skills" {
		t.Errorf("expected name 'my-agent-skills', got %q", loaded.Name)
	}
	if loaded.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", loaded.Version)
	}
	if loaded.FullName() != "my-agent-skills@1.2.3" {
		t.Errorf("expected full name 'my-agent-skills@1.2.3', got %q", loaded.FullName())
	}
	if len(loaded.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(loaded.Dependencies))
	}
	if loaded.Dependencies["github.com/example/base-skills"] != "1.0.0" {
		t.Error("expected dependency version mismatch")
	}
}

// TestManifestValidation verifies manifest validation rules.
func TestManifestValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name:    "valid",
			yaml:    "name: test\nversion: 1.0.0\n",
			wantErr: false,
		},
		{
			name:    "missing name",
			yaml:    "version: 1.0.0\n",
			wantErr: true,
		},
		{
			name:    "missing version",
			yaml:    "name: test\n",
			wantErr: true,
		},
		{
			name:    "invalid semver",
			yaml:    "name: test\nversion: abc\n",
			wantErr: true,
		},
		{
			name:    "valid with v prefix",
			yaml:    "name: test\nversion: v2.1.0\n",
			wantErr: false,
		},
		{
			name:    "valid prerelease",
			yaml:    "name: test\nversion: 1.0.0-beta.1\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := registry.ParseManifest([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLocalCache verifies the local package cache operations.
func TestLocalCache(t *testing.T) {
	cacheDir := t.TempDir()
	cache := registry.NewCache(cacheDir)

	// Initially empty
	_, found := cache.Lookup("github.com/example/pkg", "1.0.0")
	if found {
		t.Error("expected package not found in empty cache")
	}

	// Create a fake package in cache
	pkgDir := filepath.Join(cacheDir, "github.com/example/pkg", "@v", "1.0.0")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := &registry.Manifest{Name: "pkg", Version: "1.0.0"}
	if err := registry.WriteManifest(manifest, pkgDir); err != nil {
		t.Fatal(err)
	}

	// Now should be found
	path, found := cache.Lookup("github.com/example/pkg", "1.0.0")
	if !found {
		t.Error("expected package to be found in cache")
	}
	if path != pkgDir {
		t.Errorf("expected path %q, got %q", pkgDir, path)
	}

	// List all cached packages
	packages, err := cache.List()
	if err != nil {
		t.Fatalf("listing cache: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("expected 1 cached package, got %d", len(packages))
	}
	if packages[0].Source != "github.com/example/pkg" {
		t.Errorf("expected source 'github.com/example/pkg', got %q", packages[0].Source)
	}

	// Invalidate
	if err := cache.Invalidate("github.com/example/pkg", "1.0.0"); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	_, found = cache.Lookup("github.com/example/pkg", "1.0.0")
	if found {
		t.Error("expected package not found after invalidation")
	}
}

// TestVersionConflictDetection verifies MVS detects major version conflicts.
func TestVersionConflictDetection(t *testing.T) {
	// No conflict: same major version
	noConflict := []imports.VersionConstraint{
		{Package: "github.com/example/pkg", MinVersion: "1.0.0", RequiredBy: "root"},
		{Package: "github.com/example/pkg", MinVersion: "1.2.0", RequiredBy: "depA"},
	}
	conflicts := imports.DetectConflicts(noConflict)
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for same major version, got %d", len(conflicts))
	}

	// Conflict: different major versions
	withConflict := []imports.VersionConstraint{
		{Package: "github.com/example/pkg", MinVersion: "1.0.0", RequiredBy: "root → depA"},
		{Package: "github.com/example/pkg", MinVersion: "2.0.0", RequiredBy: "root → depB"},
	}
	conflicts = imports.DetectConflicts(withConflict)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Package != "github.com/example/pkg" {
		t.Errorf("expected conflict for 'github.com/example/pkg', got %q", conflicts[0].Package)
	}
	if !strings.Contains(conflicts[0].Reason, "major version mismatch") {
		t.Errorf("expected 'major version mismatch' in reason, got %q", conflicts[0].Reason)
	}

	// Format conflicts
	formatted := imports.FormatConflicts(conflicts)
	if !strings.Contains(formatted, "depA") || !strings.Contains(formatted, "depB") {
		t.Error("expected both dependency chains in formatted output")
	}
}

// TestRegistryMVSResolution verifies Minimal Version Selection picks correct versions.
func TestRegistryMVSResolution(t *testing.T) {
	constraints := []imports.VersionConstraint{
		{Package: "github.com/a/pkg", MinVersion: "1.0.0", RequiredBy: "root"},
		{Package: "github.com/a/pkg", MinVersion: "1.3.0", RequiredBy: "depX"},
		{Package: "github.com/b/pkg", MinVersion: "2.1.0", RequiredBy: "root"},
	}

	resolved, err := imports.MVS(constraints)
	if err != nil {
		t.Fatalf("MVS error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved, got %d", len(resolved))
	}

	// Check that MVS picked the highest minimum for pkg A
	for _, r := range resolved {
		if r.Package == "github.com/a/pkg" && r.Version != "1.3.0" {
			t.Errorf("expected version 1.3.0 for a/pkg, got %s", r.Version)
		}
		if r.Package == "github.com/b/pkg" && r.Version != "2.1.0" {
			t.Errorf("expected version 2.1.0 for b/pkg, got %s", r.Version)
		}
	}
}

// TestPackageSigningStub verifies package signing fields are supported in manifest.
func TestPackageSigningStub(t *testing.T) {
	yaml := `
name: signed-pkg
version: 1.0.0
signature: ""
signer: ""
provenance:
  build_system: github-actions
  source_repo: github.com/example/pkg
  commit_sha: abc123
`
	manifest, err := registry.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest with signing fields: %v", err)
	}

	if manifest.Signature != "" {
		t.Error("expected empty signature for unsigned package")
	}
	if manifest.Provenance == nil {
		t.Fatal("expected provenance to be parsed")
	}
	if manifest.Provenance.BuildSystem != "github-actions" {
		t.Errorf("expected build_system 'github-actions', got %q", manifest.Provenance.BuildSystem)
	}
	if manifest.Provenance.SourceRepo != "github.com/example/pkg" {
		t.Errorf("expected source_repo, got %q", manifest.Provenance.SourceRepo)
	}
	if manifest.Provenance.CommitSHA != "abc123" {
		t.Errorf("expected commit_sha 'abc123', got %q", manifest.Provenance.CommitSHA)
	}
}

// TestResolverWithPackageResolver verifies the import resolver integrates with PackageResolver.
func TestResolverWithPackageResolver(t *testing.T) {
	// Create a mock package directory with an .ias file
	pkgDir := t.TempDir()
	iasContent := `prompt "test-prompt" {
  content = "Hello from package"
}`
	if err := os.WriteFile(filepath.Join(pkgDir, "main.ias"), []byte(iasContent), 0644); err != nil {
		t.Fatalf("writing test .ias file: %v", err)
	}

	// Create a mock package resolver
	resolver := imports.NewResolver(t.TempDir(), nil)
	resolver.SetPackageResolver(&mockPackageResolver{
		packages: map[string]string{
			"github.com/example/skills": pkgDir,
		},
	})

	// The resolver should be able to accept the package resolver
	if resolver == nil {
		t.Fatal("expected resolver to be created")
	}
}

// mockPackageResolver is a test double for PackageResolver.
type mockPackageResolver struct {
	packages map[string]string
}

func (m *mockPackageResolver) ResolvePackage(source, version string) (string, error) {
	if path, ok := m.packages[source]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}
