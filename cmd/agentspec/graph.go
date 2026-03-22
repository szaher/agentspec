package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaher/agentspec/internal/ast"
	"github.com/szaher/agentspec/internal/graph"
	"github.com/szaher/agentspec/internal/parser"
)

func newGraphCmd() *cobra.Command {
	var (
		format  string
		port    int
		open    bool
		noOpen  bool
		theme   string
		output  string
		noFiles bool
		noOrph  bool
	)

	cmd := &cobra.Command{
		Use:   "graph [files...]",
		Short: "Visualize .ias files as an interactive dependency graph",
		Long: `Parse one or more .ias files (or a directory) and render
an interactive dependency graph showing all entities and relationships.

Output formats:
  web     Interactive web UI served on localhost (default)
  dot     Graphviz DOT format
  mermaid Mermaid markdown format`,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveFiles(args)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				return fmt.Errorf("no .ias files found")
			}

			astFiles, parseErrors := parseGraphFiles(files)
			if len(astFiles) == 0 {
				return fmt.Errorf("no valid .ias files to graph")
			}

			g := graph.Extract(astFiles)
			g.Errors = parseErrors

			if len(astFiles) > 1 {
				graph.AddFileNodes(g, astFiles)
			}

			if noFiles {
				graph.FilterFiles(g)
			}
			if noOrph {
				graph.FilterOrphans(g)
			}
			graph.ComputeStats(g)

			if noOpen {
				open = false
			}

			switch format {
			case "web":
				addr := fmt.Sprintf("http://127.0.0.1:%d", port)
				if open {
					go func() { _ = graph.OpenBrowser(addr) }()
				}
				return graph.Serve(g, port, theme)
			case "dot":
				out := graph.RenderDOT(g)
				return writeGraphOutput(out, output)
			case "mermaid":
				out := graph.RenderMermaid(g)
				return writeGraphOutput(out, output)
			default:
				return fmt.Errorf("unknown format: %s (use web, dot, or mermaid)", format)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "web", "Output format (web|dot|mermaid)")
	cmd.Flags().IntVar(&port, "port", 8686, "Port for web server")
	cmd.Flags().BoolVar(&open, "open", true, "Auto-open browser (web format)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Do not auto-open browser")
	cmd.Flags().StringVar(&theme, "theme", "dark", "Web UI theme (dark|light)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (dot/mermaid formats)")
	cmd.Flags().BoolVar(&noFiles, "no-files", false, "Remove file-type nodes from graph")
	cmd.Flags().BoolVar(&noOrph, "no-orphans", false, "Remove nodes with zero edges")

	return cmd
}

func parseGraphFiles(files []string) ([]*ast.File, []string) {
	var astFiles []*ast.File
	var parseErrors []string
	visited := map[string]bool{}

	var parseOne func(path string)
	parseOne = func(path string) {
		// Resolve symlinks for cycle detection
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			realPath = path
		}
		if visited[realPath] {
			return
		}
		visited[realPath] = true

		input, err := os.ReadFile(path)
		if err != nil {
			parseErrors = append(parseErrors, path+": "+err.Error())
			return
		}

		f, errs := parser.Parse(string(input), path)
		if errs != nil {
			for _, e := range errs {
				parseErrors = append(parseErrors, fmt.Sprintf("%s:%d: %s", path, e.Line, e.Message))
			}
			return
		}

		astFiles = append(astFiles, f)

		// Resolve imports recursively
		if f.Package != nil {
			for _, imp := range f.Package.Imports {
				impPath := imp.Path
				if !filepath.IsAbs(impPath) {
					impPath = filepath.Join(filepath.Dir(path), impPath)
				}
				parseOne(impPath)
			}
		}
		for _, stmt := range f.Statements {
			if imp, ok := stmt.(*ast.Import); ok {
				impPath := imp.Path
				if !filepath.IsAbs(impPath) {
					impPath = filepath.Join(filepath.Dir(path), impPath)
				}
				parseOne(impPath)
			}
		}
	}

	for _, file := range files {
		parseOne(file)
	}

	for _, e := range parseErrors {
		fmt.Fprintln(os.Stderr, "warning: "+e)
	}

	return astFiles, parseErrors
}

func writeGraphOutput(content, output string) error {
	if output == "" {
		_, err := fmt.Print(content)
		return err
	}
	return os.WriteFile(output, []byte(content), 0o644)
}
