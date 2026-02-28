package imports

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// LockFile represents the contents of .agentspec.lock.
// It records resolved versions and content hashes for reproducible builds.
type LockFile struct {
	Version      string        `json:"version"`
	Dependencies []LockedDep   `json:"dependencies"`
}

// LockedDep records a single resolved dependency.
type LockedDep struct {
	Source  string `json:"source"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash"`
	Path    string `json:"path"`
}

const lockFileName = ".agentspec.lock"

// ReadLockFile reads and parses a lock file from the given directory.
// Returns nil if the lock file does not exist.
func ReadLockFile(dir string) (*LockFile, error) {
	path := lockFilePath(dir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}

	return &lf, nil
}

// WriteLockFile writes the lock file to the given directory.
func WriteLockFile(dir string, lf *LockFile) error {
	// Sort dependencies for deterministic output
	sort.Slice(lf.Dependencies, func(i, j int) bool {
		return lf.Dependencies[i].Source < lf.Dependencies[j].Source
	})

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling lock file: %w", err)
	}

	path := lockFilePath(dir)
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	return nil
}

// GenerateLockFile creates a lock file from resolved imports.
func GenerateLockFile(resolved []*ResolvedImport) *LockFile {
	seen := make(map[string]bool)
	var deps []LockedDep

	for _, ri := range resolved {
		if seen[ri.Source] {
			continue
		}
		seen[ri.Source] = true

		deps = append(deps, LockedDep{
			Source:  ri.Source,
			Version: ri.Version,
			Hash:    ri.Hash,
			Path:    ri.Path,
		})
	}

	return &LockFile{
		Version:      "1",
		Dependencies: deps,
	}
}

// ValidateLockFile checks that all resolved imports match the lock file.
// Returns a list of mismatches (empty if everything matches).
func ValidateLockFile(lf *LockFile, resolved []*ResolvedImport) []string {
	if lf == nil {
		return nil
	}

	// Build lookup from lock file
	locked := make(map[string]LockedDep)
	for _, dep := range lf.Dependencies {
		locked[dep.Source] = dep
	}

	var mismatches []string
	for _, ri := range resolved {
		if ld, ok := locked[ri.Source]; ok {
			if ld.Hash != "" && ld.Hash != ri.Hash {
				mismatches = append(mismatches, fmt.Sprintf(
					"%s: hash mismatch (locked: %s, resolved: %s)",
					ri.Source, ld.Hash, ri.Hash,
				))
			}
			if ld.Version != "" && ri.Version != "" && ld.Version != ri.Version {
				mismatches = append(mismatches, fmt.Sprintf(
					"%s: version mismatch (locked: %s, resolved: %s)",
					ri.Source, ld.Version, ri.Version,
				))
			}
		}
	}

	return mismatches
}

func lockFilePath(dir string) string {
	return dir + "/" + lockFileName
}
