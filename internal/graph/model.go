// Package graph extracts and renders dependency graphs from AgentSpec AST files.
package graph

// GraphNode represents a single entity extracted from a parsed .ias file.
type GraphNode struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	File       string            `json:"file,omitempty"`
	Line       int               `json:"line,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// GraphEdge represents a directed relationship between two entities.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
	Style  string `json:"style,omitempty"`
}

// Graph is the top-level container for visualization data.
type Graph struct {
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
	Package PackageInfo `json:"package,omitempty"`
	Files   []string    `json:"files"`
	Stats   GraphStats  `json:"stats"`
	Errors  []string    `json:"errors,omitempty"`
}

// PackageInfo holds package metadata from the .ias header.
type PackageInfo struct {
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// GraphStats holds aggregate counts for the graph.
type GraphStats struct {
	NodeCount  int            `json:"node_count"`
	EdgeCount  int            `json:"edge_count"`
	FileCount  int            `json:"file_count"`
	TypeCounts map[string]int `json:"type_counts"`
}
