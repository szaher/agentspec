// Package registry implements the AgentSpec package registry and resolution.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest represents an agentpack.yaml package manifest.
type Manifest struct {
	Name         string            `yaml:"name" json:"name"`
	Version      string            `yaml:"version" json:"version"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Author       string            `yaml:"author,omitempty" json:"author,omitempty"`
	License      string            `yaml:"license,omitempty" json:"license,omitempty"`
	AgentSpec    string            `yaml:"agentspec,omitempty" json:"agentspec,omitempty"`
	Dependencies map[string]string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Exports      []string          `yaml:"exports,omitempty" json:"exports,omitempty"`
	Signature    string            `yaml:"signature,omitempty" json:"signature,omitempty"`
	Signer       string            `yaml:"signer,omitempty" json:"signer,omitempty"`
	Provenance   *Provenance       `yaml:"provenance,omitempty" json:"provenance,omitempty"`
}

// Provenance holds build provenance metadata.
type Provenance struct {
	BuildSystem string `yaml:"build_system,omitempty" json:"build_system,omitempty"`
	SourceRepo  string `yaml:"source_repo,omitempty" json:"source_repo,omitempty"`
	CommitSHA   string `yaml:"commit_sha,omitempty" json:"commit_sha,omitempty"`
}

// ManifestFile is the default manifest filename.
const ManifestFile = "agentpack.yaml"

// ReadManifest reads and parses an agentpack.yaml manifest.
func ReadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	return ParseManifest(data)
}

// ParseManifest parses manifest data from YAML bytes.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// WriteManifest writes a manifest to the given directory.
func WriteManifest(m *Manifest, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, ManifestFile), data, 0644)
}

// Validate checks that the manifest has required fields.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("manifest: name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest: version is required")
	}
	if !isValidSemver(m.Version) {
		return fmt.Errorf("manifest: invalid semver version %q", m.Version)
	}
	for dep, ver := range m.Dependencies {
		if dep == "" {
			return fmt.Errorf("manifest: empty dependency name")
		}
		if ver == "" {
			return fmt.Errorf("manifest: dependency %q has no version constraint", dep)
		}
	}
	return nil
}

// FullName returns the package name with version.
func (m *Manifest) FullName() string {
	return m.Name + "@" + m.Version
}

// isValidSemver checks if a version string looks like semver (vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH).
func isValidSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		// Allow pre-release suffix on last part
		base := strings.SplitN(p, "-", 2)[0]
		if base == "" {
			return false
		}
		for _, c := range base {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
