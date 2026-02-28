// Package imports provides import resolution, dependency graph management,
// lock file handling, and Minimal Version Selection for IntentLang packages.
package imports

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/szaher/designs/agentz/internal/ast"
	"github.com/szaher/designs/agentz/internal/parser"
)

// ResolvedImport holds the result of resolving a single import statement.
type ResolvedImport struct {
	Source      string   // original import path
	Kind       string   // "local" or "package"
	Alias      string   // import alias
	Version    string   // version constraint (packages only)
	Path       string   // absolute resolved path
	Hash       string   // content hash (SHA-256)
	File       *ast.File // parsed AST
	Resources  []string // resource names provided by this import
}

// PackageResolver resolves versioned package imports to local paths.
// This interface is implemented by the registry client.
type PackageResolver interface {
	// ResolvePackage resolves a package source and version to a local directory path.
	ResolvePackage(source, version string) (string, error)
}

// Resolver resolves import statements to parsed AST files.
type Resolver struct {
	// baseDir is the directory of the root file being compiled.
	baseDir string
	// cache prevents re-parsing the same file.
	cache map[string]*ResolvedImport
	// searchPaths for package imports (e.g., ~/.agentspec/packages/)
	searchPaths []string
	// visited tracks files being processed to detect cycles in resolution
	visited map[string]bool
	// pkgResolver handles package resolution via registry
	pkgResolver PackageResolver
}

// NewResolver creates a new import resolver rooted at the given directory.
func NewResolver(baseDir string, searchPaths []string) *Resolver {
	return &Resolver{
		baseDir:     baseDir,
		cache:       make(map[string]*ResolvedImport),
		searchPaths: searchPaths,
		visited:     make(map[string]bool),
	}
}

// SetPackageResolver sets the package resolver for registry-based imports.
func (r *Resolver) SetPackageResolver(pr PackageResolver) {
	r.pkgResolver = pr
}

// ResolveAll resolves all imports from a parsed file and returns the resolved imports.
// It recursively resolves transitive imports.
func (r *Resolver) ResolveAll(f *ast.File) ([]*ResolvedImport, error) {
	// Collect import statements from the file's top-level statements
	var importStmts []*ast.Import
	for _, stmt := range f.Statements {
		if imp, ok := stmt.(*ast.Import); ok {
			importStmts = append(importStmts, imp)
		}
	}

	// Also check Package.Imports if populated
	if f.Package != nil && len(f.Package.Imports) > 0 {
		importStmts = append(importStmts, f.Package.Imports...)
	}

	if len(importStmts) == 0 {
		return nil, nil
	}

	var resolved []*ResolvedImport
	for _, imp := range importStmts {
		ri, err := r.resolveImport(imp, r.baseDir)
		if err != nil {
			return nil, fmt.Errorf("resolving import %q: %w", imp.Path, err)
		}
		resolved = append(resolved, ri)

		// Recursively resolve transitive imports
		transitive, err := r.resolveTransitive(ri)
		if err != nil {
			return nil, fmt.Errorf("resolving transitive imports from %q: %w", imp.Path, err)
		}
		resolved = append(resolved, transitive...)
	}

	return resolved, nil
}

// resolveImport resolves a single import to a file on disk.
func (r *Resolver) resolveImport(imp *ast.Import, fromDir string) (*ResolvedImport, error) {
	importPath := imp.Path

	// Check cache
	cacheKey := importPath
	if imp.Version != "" {
		cacheKey = importPath + "@" + imp.Version
	}
	if cached, ok := r.cache[cacheKey]; ok {
		return cached, nil
	}

	// Determine if local or package import
	kind := classifyImport(importPath)

	var absPath string
	var err error

	switch kind {
	case "local":
		absPath, err = r.resolveLocalPath(importPath, fromDir)
		if err != nil {
			return nil, err
		}
	case "package":
		// Try registry-based resolution first
		if r.pkgResolver != nil {
			pkgDir, resolveErr := r.pkgResolver.ResolvePackage(importPath, imp.Version)
			if resolveErr == nil {
				absPath, err = r.findEntryPoint(pkgDir)
				if err != nil {
					return nil, fmt.Errorf("package %q resolved but no entry point found: %w", importPath, err)
				}
				break
			}
			// Fall through to search paths if registry resolution fails
		}
		absPath, err = r.resolvePackagePath(importPath, imp.Version)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown import kind for %q", importPath)
	}

	// Read and parse the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", absPath, err)
	}

	f, parseErrs := parser.Parse(string(content), absPath)
	if parseErrs != nil {
		return nil, fmt.Errorf("parsing %s: %v", absPath, parseErrs)
	}

	// Collect resource names
	var resources []string
	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Prompt:
			resources = append(resources, "Prompt/"+s.Name)
		case *ast.Skill:
			resources = append(resources, "Skill/"+s.Name)
		case *ast.Agent:
			resources = append(resources, "Agent/"+s.Name)
		}
	}

	hash := computeContentHash(content)

	ri := &ResolvedImport{
		Source:    importPath,
		Kind:     kind,
		Alias:    imp.Alias,
		Version:  imp.Version,
		Path:     absPath,
		Hash:     hash,
		File:     f,
		Resources: resources,
	}

	r.cache[cacheKey] = ri
	return ri, nil
}

// resolveTransitive resolves imports from an already-resolved import file.
func (r *Resolver) resolveTransitive(ri *ResolvedImport) ([]*ResolvedImport, error) {
	if ri.File == nil {
		return nil, nil
	}

	// Prevent infinite recursion on circular imports
	if r.visited[ri.Path] {
		return nil, nil
	}
	r.visited[ri.Path] = true

	// Collect import statements from the resolved file
	var importStmts []*ast.Import
	for _, stmt := range ri.File.Statements {
		if imp, ok := stmt.(*ast.Import); ok {
			importStmts = append(importStmts, imp)
		}
	}
	if ri.File.Package != nil {
		importStmts = append(importStmts, ri.File.Package.Imports...)
	}

	if len(importStmts) == 0 {
		return nil, nil
	}

	fromDir := filepath.Dir(ri.Path)
	var resolved []*ResolvedImport

	for _, imp := range importStmts {
		tri, err := r.resolveImport(imp, fromDir)
		if err != nil {
			return nil, fmt.Errorf("resolving %q from %q: %w", imp.Path, ri.Source, err)
		}
		resolved = append(resolved, tri)

		// Continue recursively
		transitive, err := r.resolveTransitive(tri)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, transitive...)
	}

	return resolved, nil
}

// findEntryPoint finds the main .ias entry point in a package directory.
func (r *Resolver) findEntryPoint(pkgDir string) (string, error) {
	candidates := []string{"main.ias", "index.ias"}
	for _, name := range candidates {
		path := filepath.Join(pkgDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	// Try any .ias file
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", fmt.Errorf("reading package dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ias") {
			return filepath.Join(pkgDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .ias files in %s", pkgDir)
}

// resolveLocalPath resolves a relative import path to an absolute path.
func (r *Resolver) resolveLocalPath(importPath, fromDir string) (string, error) {
	// Handle relative paths
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		absPath := filepath.Join(fromDir, importPath)
		absPath = filepath.Clean(absPath)

		// Try with .ias extension if not already present
		if !strings.HasSuffix(absPath, ".ias") {
			withExt := absPath + ".ias"
			if _, err := os.Stat(withExt); err == nil {
				return withExt, nil
			}
			// Try as directory with index.ias
			indexPath := filepath.Join(absPath, "index.ias")
			if _, err := os.Stat(indexPath); err == nil {
				return indexPath, nil
			}
		}

		if _, err := os.Stat(absPath); err != nil {
			return "", fmt.Errorf("import file not found: %s", absPath)
		}
		return absPath, nil
	}

	// Bare path — resolve relative to base dir
	absPath := filepath.Join(fromDir, importPath)
	absPath = filepath.Clean(absPath)

	if !strings.HasSuffix(absPath, ".ias") {
		withExt := absPath + ".ias"
		if _, err := os.Stat(withExt); err == nil {
			return withExt, nil
		}
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("import file not found: %s", absPath)
	}
	return absPath, nil
}

// resolvePackagePath resolves a versioned package import to an absolute path.
func (r *Resolver) resolvePackagePath(importPath, version string) (string, error) {
	// Search in search paths for the package
	for _, searchDir := range r.searchPaths {
		// Package layout: <searchDir>/<package-name>@<version>/
		pkgDir := filepath.Join(searchDir, importPath)
		if version != "" {
			pkgDir = filepath.Join(searchDir, importPath+"@"+version)
		}

		// Look for main entry point
		candidates := []string{
			filepath.Join(pkgDir, "main.ias"),
			filepath.Join(pkgDir, "index.ias"),
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}

		// Try without version suffix
		if version != "" {
			pkgDir = filepath.Join(searchDir, importPath)
			for _, name := range []string{"main.ias", "index.ias"} {
				candidate := filepath.Join(pkgDir, name)
				if _, err := os.Stat(candidate); err == nil {
					return candidate, nil
				}
			}
		}
	}

	return "", fmt.Errorf("package %q (version %s) not found in search paths", importPath, version)
}

// classifyImport determines whether an import is local or a package reference.
func classifyImport(path string) string {
	// Local: starts with ./ or ../ or ends with .ias
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return "local"
	}
	if strings.HasSuffix(path, ".ias") {
		return "local"
	}
	// Package: contains a / that looks like a domain (e.g., github.com/...)
	if strings.Contains(path, "/") && strings.Contains(path, ".") {
		return "package"
	}
	// Default to local for simple names
	return "local"
}

// MergeImports merges resources from resolved imports into the main AST file.
// Resources are prefixed with the import alias if one is set.
func MergeImports(mainFile *ast.File, resolved []*ResolvedImport) *ast.File {
	// Track which resources we've already added (dedup by path)
	seen := make(map[string]bool)

	for _, ri := range resolved {
		if seen[ri.Path] {
			continue
		}
		seen[ri.Path] = true

		for _, stmt := range ri.File.Statements {
			// Skip package declarations — they're metadata for the imported file
			switch stmt.(type) {
			case *ast.Package:
				continue
			case *ast.Import:
				continue
			}
			mainFile.Statements = append(mainFile.Statements, stmt)
		}
	}

	return mainFile
}
