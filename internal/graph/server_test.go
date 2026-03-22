package graph

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeAPIGraph(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:test", Type: "agent", Name: "test"},
		},
		Edges: []GraphEdge{},
		Files: []string{"test.ias"},
		Stats: GraphStats{NodeCount: 1, EdgeCount: 0, FileCount: 1, TypeCounts: map[string]int{"agent": 1}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(g)
	})

	req := httptest.NewRequest("GET", "/api/graph", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result Graph
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(result.Nodes))
	}
	if result.Nodes[0].ID != "agent:test" {
		t.Errorf("expected agent:test, got %s", result.Nodes[0].ID)
	}
	if result.Stats.NodeCount != 1 {
		t.Errorf("expected node_count 1, got %d", result.Stats.NodeCount)
	}
}

func TestServeHTMLIndex(t *testing.T) {
	mux := http.NewServeMux()

	subFS, err := fs.Sub(WebFS, "web")
	if err != nil {
		t.Fatalf("failed to get web FS: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "AgentSpec Graph") {
		t.Error("expected HTML to contain 'AgentSpec Graph'")
	}
	if !strings.Contains(body, "<html") {
		t.Error("expected HTML content")
	}
}
