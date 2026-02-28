package imports

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// Graph represents a dependency graph of import relationships.
type Graph struct {
	// nodes maps file path to its node
	nodes map[string]*GraphNode
}

// GraphNode represents a single file in the dependency graph.
type GraphNode struct {
	Path    string
	Hash    string
	Imports []string // paths of direct dependencies
}

// NewGraph creates an empty dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*GraphNode),
	}
}

// AddNode adds a file to the dependency graph.
func (g *Graph) AddNode(path, hash string, imports []string) {
	g.nodes[path] = &GraphNode{
		Path:    path,
		Hash:    hash,
		Imports: imports,
	}
}

// AddFromResolved populates the graph from resolved imports.
func (g *Graph) AddFromResolved(rootPath string, resolved []*ResolvedImport) {
	// Add root node
	rootImports := make([]string, 0)
	for _, ri := range resolved {
		// Only add direct imports (not transitive)
		if ri.File != nil {
			rootImports = append(rootImports, ri.Path)
		}
	}
	g.AddNode(rootPath, "", rootImports)

	// Add resolved import nodes
	for _, ri := range resolved {
		var deps []string
		if ri.File != nil && ri.File.Package != nil {
			for _, imp := range ri.File.Package.Imports {
				// Find the resolved path for this import
				for _, other := range resolved {
					if other.Source == imp.Path {
						deps = append(deps, other.Path)
						break
					}
				}
			}
		}
		g.AddNode(ri.Path, ri.Hash, deps)
	}
}

// DetectCycles detects circular dependencies using Tarjan's SCC algorithm.
// Returns a list of cycles found, where each cycle is a list of file paths.
func (g *Graph) DetectCycles() [][]string {
	t := &tarjan{
		graph:   g,
		index:   0,
		stack:   nil,
		onStack: make(map[string]bool),
		indices: make(map[string]int),
		lowlink: make(map[string]int),
	}

	for path := range g.nodes {
		if _, visited := t.indices[path]; !visited {
			t.strongconnect(path)
		}
	}

	// Filter SCCs to only include those with more than one node (actual cycles)
	// or self-loops
	var cycles [][]string
	for _, scc := range t.sccs {
		if len(scc) > 1 {
			cycles = append(cycles, scc)
		} else if len(scc) == 1 {
			// Check for self-loop
			node := g.nodes[scc[0]]
			if node != nil {
				for _, dep := range node.Imports {
					if dep == scc[0] {
						cycles = append(cycles, scc)
						break
					}
				}
			}
		}
	}

	return cycles
}

// TopologicalSort returns nodes in dependency order (dependencies first).
// Returns an error if cycles are detected.
func (g *Graph) TopologicalSort() ([]string, error) {
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		return nil, fmt.Errorf("circular dependencies detected: %v", cycles[0])
	}

	var sorted []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool) // for safety

	var visit func(path string) error
	visit = func(path string) error {
		if visited[path] {
			return nil
		}
		if visiting[path] {
			return fmt.Errorf("cycle detected at %s", path)
		}
		visiting[path] = true

		node := g.nodes[path]
		if node != nil {
			for _, dep := range node.Imports {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visiting[path] = false
		visited[path] = true
		sorted = append(sorted, path)
		return nil
	}

	// Sort keys for deterministic output
	paths := make([]string, 0, len(g.nodes))
	for path := range g.nodes {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		if err := visit(path); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// DependencyChain returns the chain of dependencies from root to target.
func (g *Graph) DependencyChain(root, target string) []string {
	var chain []string
	visited := make(map[string]bool)

	var dfs func(current string) bool
	dfs = func(current string) bool {
		if visited[current] {
			return false
		}
		visited[current] = true
		chain = append(chain, current)

		if current == target {
			return true
		}

		node := g.nodes[current]
		if node != nil {
			for _, dep := range node.Imports {
				if dfs(dep) {
					return true
				}
			}
		}

		chain = chain[:len(chain)-1]
		return false
	}

	dfs(root)
	return chain
}

// tarjan implements Tarjan's strongly connected components algorithm.
type tarjan struct {
	graph   *Graph
	index   int
	stack   []string
	onStack map[string]bool
	indices map[string]int
	lowlink map[string]int
	sccs    [][]string
}

func (t *tarjan) strongconnect(v string) {
	t.indices[v] = t.index
	t.lowlink[v] = t.index
	t.index++
	t.stack = append(t.stack, v)
	t.onStack[v] = true

	node := t.graph.nodes[v]
	if node != nil {
		for _, w := range node.Imports {
			if _, visited := t.indices[w]; !visited {
				t.strongconnect(w)
				if t.lowlink[w] < t.lowlink[v] {
					t.lowlink[v] = t.lowlink[w]
				}
			} else if t.onStack[w] {
				if t.indices[w] < t.lowlink[v] {
					t.lowlink[v] = t.indices[w]
				}
			}
		}
	}

	if t.lowlink[v] == t.indices[v] {
		var scc []string
		for {
			w := t.stack[len(t.stack)-1]
			t.stack = t.stack[:len(t.stack)-1]
			t.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		t.sccs = append(t.sccs, scc)
	}
}

// computeContentHash computes SHA-256 hash of content.
func computeContentHash(content []byte) string {
	h := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(h[:])
}
