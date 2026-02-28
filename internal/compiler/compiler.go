package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/ast"
	"github.com/szaher/designs/agentz/internal/compiler/targets"
	"github.com/szaher/designs/agentz/internal/imports"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/validate"
)

// CompileOptions configures the compilation process.
type CompileOptions struct {
	Target        string // "standalone", "crewai", etc.
	OutputDir     string // output directory
	Platform      string // e.g. "linux/amd64"
	Name          string // binary/project name
	EmbedFrontend bool   // embed frontend assets
	Verbose       bool   // verbose output
	Version       string // version to embed
	JSONOutput    bool   // output JSON result
}

// CompileResult is the output of the compilation pipeline.
type CompileResult struct {
	Status          string   `json:"status"`
	Target          string   `json:"target"`
	Platform        string   `json:"platform"`
	OutputPath      string   `json:"output_path"`
	SizeBytes       int64    `json:"size_bytes"`
	ContentHash     string   `json:"content_hash"`
	SourceHash      string   `json:"source_hash"`
	Agents          []string `json:"agents"`
	ConfigRef       string   `json:"config_ref"`
	Warnings        []string `json:"warnings,omitempty"`
	CompileTimeMS   int64    `json:"compilation_time_ms"`
	FilesProcessed  int      `json:"files_processed"`
	ImportsResolved int      `json:"imports_resolved"`
}

// Compile is the core compilation orchestrator.
// Pipeline: parse .ias → validate → lower to IR → select target → invoke → produce artifact.
func Compile(files []string, opts CompileOptions) (*CompileResult, error) {
	startTime := time.Now()

	if len(files) == 0 {
		return nil, fmt.Errorf("no input files specified")
	}

	if opts.Target == "" {
		opts.Target = "standalone"
	}

	if opts.Platform == "" {
		opts.Platform = CurrentPlatform()
	}

	if err := ValidatePlatform(opts.Platform); err != nil {
		return nil, err
	}

	if opts.Version == "" {
		opts.Version = "0.3.0"
	}

	// Phase 1: Parse all input files
	var allFiles []*parsedFile
	for _, path := range files {
		pf, err := parseFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		allFiles = append(allFiles, pf)
	}

	// Phase 2: Resolve imports
	var importsResolved int
	for _, pf := range allFiles {
		if hasImports(pf.ast) {
			baseDir := filepath.Dir(pf.path)
			resolver := imports.NewResolver(baseDir, defaultSearchPaths())

			resolved, err := resolver.ResolveAll(pf.ast)
			if err != nil {
				return nil, fmt.Errorf("import resolution in %s: %w", pf.path, err)
			}

			if len(resolved) > 0 {
				// Check for circular dependencies
				graph := imports.NewGraph()
				graph.AddFromResolved(pf.path, resolved)
				cycles := graph.DetectCycles()
				if len(cycles) > 0 {
					return nil, fmt.Errorf("circular dependency in %s: %v", pf.path, cycles[0])
				}

				// Merge imported resources into the main AST
				imports.MergeImports(pf.ast, resolved)
				importsResolved += len(resolved)
			}
		}
	}

	// Phase 3: Validate
	for _, pf := range allFiles {
		errs := validate.ValidateStructural(pf.ast)
		if len(errs) > 0 {
			return nil, fmt.Errorf("validation errors in %s: %v", pf.path, errs)
		}
	}

	// Phase 4: Lower to IR
	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no files to compile")
	}

	doc, err := ir.Lower(allFiles[0].ast)
	if err != nil {
		return nil, fmt.Errorf("lowering to IR: %w", err)
	}

	// Compute source hash
	sourceHash := computeSourceHash(allFiles)

	// Phase 5: Convert IR to RuntimeConfig
	config, err := runtime.FromIR(doc)
	if err != nil {
		return nil, fmt.Errorf("converting IR to runtime config: %w", err)
	}

	// Determine artifact name
	name := opts.Name
	if name == "" {
		name = config.PackageName
	}

	// Phase 6: Select target and compile
	var warnings []string

	switch opts.Target {
	case "standalone":
		result, err := CompileStandalone(config, opts)
		if err != nil {
			return nil, fmt.Errorf("standalone compilation: %w", err)
		}

		return &CompileResult{
			Status:          "success",
			Target:          opts.Target,
			Platform:        result.Platform,
			OutputPath:      result.OutputPath,
			SizeBytes:       result.SizeBytes,
			ContentHash:     result.ContentHash,
			SourceHash:      sourceHash,
			Agents:          result.Agents,
			ConfigRef:       result.ConfigRef,
			Warnings:        warnings,
			CompileTimeMS:   time.Since(startTime).Milliseconds(),
			FilesProcessed:  len(allFiles),
			ImportsResolved: importsResolved,
		}, nil

	default:
		// Framework targets: check built-in targets first
		target, ok := targets.Get(opts.Target)
		if !ok {
			available := append([]string{"standalone"}, targets.List()...)
			return nil, fmt.Errorf("unsupported compilation target: %q (available: %s)", opts.Target, strings.Join(available, ", "))
		}

		// Run feature gap analysis
		detectedFeatures := DetectFeatures(doc)
		featureMap := target.FeatureSupport()
		gapWarnings := AnalyzeGaps(detectedFeatures, featureMap)
		warnings = append(warnings, GapWarningsToStrings(gapWarnings)...)

		// Compile to framework target
		result, err := target.Compile(doc, name)
		if err != nil {
			return nil, fmt.Errorf("%s compilation: %w", opts.Target, err)
		}

		// Add compile warnings from the target
		for _, w := range result.Warnings {
			warnings = append(warnings, fmt.Sprintf("[%s] %s", w.Feature, w.Message))
		}

		// Apply safe zone preservation if output dir has existing files
		outputDir := opts.OutputDir
		if outputDir == "" {
			outputDir = "build"
		}
		commentPrefix := CommentPrefixForLanguage(result.Metadata.PythonVersion)
		if result.Metadata.PythonVersion != "" {
			commentPrefix = "#"
		}
		for i, f := range result.Files {
			existingPath := filepath.Join(outputDir, f.Path)
			if existingContent, err := os.ReadFile(existingPath); err == nil {
				userCode := ExtractUserCode(string(existingContent), commentPrefix)
				if len(userCode) > 0 {
					result.Files[i].Content = MergeWithUserCode(f.Content, commentPrefix, userCode)
				}
			}
		}

		// Write generated files
		if err := targets.WriteFiles(outputDir, result.Files); err != nil {
			return nil, fmt.Errorf("writing generated files: %w", err)
		}

		return &CompileResult{
			Status:          "success",
			Target:          opts.Target,
			Platform:        result.Metadata.Framework,
			OutputPath:      outputDir,
			Agents:          agentNames(doc),
			Warnings:        warnings,
			CompileTimeMS:   time.Since(startTime).Milliseconds(),
			FilesProcessed:  len(allFiles),
			ImportsResolved: importsResolved,
		}, nil
	}
}

type parsedFile struct {
	path    string
	content []byte
	ast     *ast.File
}

func parseFile(path string) (*parsedFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	f, parseErrs := parser.Parse(string(content), path)
	if parseErrs != nil {
		return nil, fmt.Errorf("parse errors: %v", parseErrs)
	}

	return &parsedFile{
		path:    path,
		content: content,
		ast:     f,
	}, nil
}

func computeSourceHash(files []*parsedFile) string {
	h := sha256.New()
	for _, f := range files {
		h.Write(f.content)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// hasImports checks whether a file has any import statements.
func hasImports(f *ast.File) bool {
	for _, stmt := range f.Statements {
		if _, ok := stmt.(*ast.Import); ok {
			return true
		}
	}
	if f.Package != nil && len(f.Package.Imports) > 0 {
		return true
	}
	return false
}

// defaultSearchPaths returns the default package search paths.
func defaultSearchPaths() []string {
	var paths []string

	// User home directory
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(home, ".agentspec", "packages"))
		paths = append(paths, filepath.Join(home, ".agentz", "packages")) // legacy
	}

	return paths
}

// agentNames extracts agent names from an IR document.
func agentNames(doc *ir.Document) []string {
	var names []string
	for _, r := range doc.Resources {
		if r.Kind == "Agent" {
			names = append(names, r.Name)
		}
	}
	return names
}
