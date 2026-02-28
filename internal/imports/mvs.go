package imports

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// VersionConstraint represents a single version requirement.
type VersionConstraint struct {
	Package    string // package name/path
	MinVersion string // minimum acceptable version (semver)
	RequiredBy string // which package requires this
}

// ResolvedVersion is the output of MVS — a single resolved version.
type ResolvedVersion struct {
	Package string `json:"package"`
	Version string `json:"version"`
}

// MVS implements Minimal Version Selection.
// Given a set of version constraints from all transitive dependencies,
// it selects the minimum version that satisfies all constraints.
func MVS(constraints []VersionConstraint) ([]ResolvedVersion, error) {
	// Group constraints by package
	byPackage := make(map[string][]VersionConstraint)
	for _, c := range constraints {
		byPackage[c.Package] = append(byPackage[c.Package], c)
	}

	var resolved []ResolvedVersion

	for pkg, cs := range byPackage {
		// Find the maximum minimum version (the minimum version that satisfies all)
		maxMin := ""
		for _, c := range cs {
			if maxMin == "" || compareSemver(c.MinVersion, maxMin) > 0 {
				maxMin = c.MinVersion
			}
		}

		if maxMin == "" {
			return nil, fmt.Errorf("no version constraint for package %q", pkg)
		}

		resolved = append(resolved, ResolvedVersion{
			Package: pkg,
			Version: maxMin,
		})
	}

	// Sort for deterministic output
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Package < resolved[j].Package
	})

	return resolved, nil
}

// ValidateConstraints checks that all constraints can be satisfied simultaneously.
// Returns an error if there's a conflict.
func ValidateConstraints(constraints []VersionConstraint) error {
	_, err := MVS(constraints)
	if err != nil {
		return err
	}
	conflicts := DetectConflicts(constraints)
	if len(conflicts) > 0 {
		return fmt.Errorf("version conflicts detected:\n%s", FormatConflicts(conflicts))
	}
	return nil
}

// VersionConflict describes an incompatible version requirement between dependencies.
type VersionConflict struct {
	Package string
	ChainA  string // dependency chain A (e.g., "root → pkgA → pkgC@1.0.0")
	ChainB  string // dependency chain B (e.g., "root → pkgB → pkgC@2.0.0")
	Reason  string // e.g., "major version mismatch"
}

// DetectConflicts checks for incompatible version requirements in the constraint set.
// MVS always picks the highest minimum version, but if two constraints require
// different major versions, this is flagged as a conflict.
func DetectConflicts(constraints []VersionConstraint) []VersionConflict {
	byPackage := make(map[string][]VersionConstraint)
	for _, c := range constraints {
		byPackage[c.Package] = append(byPackage[c.Package], c)
	}

	var conflicts []VersionConflict
	for pkg, cs := range byPackage {
		if len(cs) < 2 {
			continue
		}

		// Check for major version mismatches
		for i := 0; i < len(cs); i++ {
			for j := i + 1; j < len(cs); j++ {
				aParts := parseSemver(cs[i].MinVersion)
				bParts := parseSemver(cs[j].MinVersion)

				if aParts[0] != bParts[0] {
					conflicts = append(conflicts, VersionConflict{
						Package: pkg,
						ChainA:  fmt.Sprintf("%s requires %s@%s", cs[i].RequiredBy, pkg, cs[i].MinVersion),
						ChainB:  fmt.Sprintf("%s requires %s@%s", cs[j].RequiredBy, pkg, cs[j].MinVersion),
						Reason:  fmt.Sprintf("major version mismatch: v%d vs v%d", aParts[0], bParts[0]),
					})
				}
			}
		}
	}
	return conflicts
}

// FormatConflicts formats conflicts into a readable string.
func FormatConflicts(conflicts []VersionConflict) string {
	var sb strings.Builder
	for i, c := range conflicts {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "  - %s: %s\n", c.Package, c.Reason)
		fmt.Fprintf(&sb, "    %s\n", c.ChainA)
		fmt.Fprintf(&sb, "    %s\n", c.ChainB)
		fmt.Fprintf(&sb, "    Suggestion: align both dependencies to the same major version")
	}
	return sb.String()
}

// ExtractConstraints extracts version constraints from resolved imports.
func ExtractConstraints(resolved []*ResolvedImport, requiredBy string) []VersionConstraint {
	var constraints []VersionConstraint
	for _, ri := range resolved {
		if ri.Kind == "package" && ri.Version != "" {
			constraints = append(constraints, VersionConstraint{
				Package:    ri.Source,
				MinVersion: ri.Version,
				RequiredBy: requiredBy,
			})
		}
	}
	return constraints
}

// compareSemver compares two semver strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareSemver(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// parseSemver splits a version string into [major, minor, patch].
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	// Strip pre-release suffix
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}

	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}
