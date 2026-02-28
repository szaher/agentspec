// Package frontend provides the built-in web UI for AgentSpec agents.
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web
var webFS embed.FS

// Handler serves the embedded frontend static files.
type Handler struct {
	fileServer http.Handler
	prefix     string
}

// NewHandler creates a frontend handler that serves embedded static files.
// The prefix is the URL path prefix (e.g., "/ui/").
func NewHandler(prefix string) *Handler {
	// Strip the "web" directory from the embedded FS
	subFS, _ := fs.Sub(webFS, "web")

	return &Handler{
		fileServer: http.FileServer(http.FS(subFS)),
		prefix:     prefix,
	}
}

// ServeHTTP serves embedded static files with SPA fallback to index.html.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip the prefix for file lookup
	path := strings.TrimPrefix(r.URL.Path, h.prefix)
	if path == "" || path == "/" {
		path = "index.html"
	}

	// Try to open the requested file from the embedded FS
	subFS, _ := fs.Sub(webFS, "web")
	if _, err := fs.Stat(subFS, path); err != nil {
		// File not found — serve index.html for SPA routing
		r.URL.Path = h.prefix + "/"
	}

	// Strip prefix and serve
	http.StripPrefix(h.prefix, h.fileServer).ServeHTTP(w, r)
}

// Mount registers the frontend handler on the given ServeMux.
// It mounts on both "/" and "/ui/" paths.
func Mount(mux *http.ServeMux, h *Handler) {
	mux.Handle("/ui/", h)
	mux.Handle("/", h)
}
