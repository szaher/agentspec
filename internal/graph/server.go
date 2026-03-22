package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/signal"
	"syscall"
)

// Serve starts an HTTP server on 127.0.0.1:port serving the graph web UI.
// It blocks until SIGINT/SIGTERM is received, then shuts down gracefully.
func Serve(g *Graph, port int, theme string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	// API endpoint
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		if err := json.NewEncoder(w).Encode(g); err != nil {
			http.Error(w, `{"error":"failed to encode graph"}`, http.StatusInternalServerError)
		}
	})

	// Serve embedded web assets
	subFS, err := fs.Sub(WebFS, "web")
	if err != nil {
		return fmt.Errorf("failed to access embedded web assets: %w", err)
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Pass theme as query parameter for the HTML to read
		if r.URL.Path == "/" && r.URL.RawQuery == "" && theme != "" {
			http.Redirect(w, r, "/?theme="+theme, http.StatusTemporaryRedirect)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use. Try --port %d", port, port+1)
	}

	server := &http.Server{Handler: mux}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	_, _ = fmt.Printf("Serving graph at http://%s\nPress Ctrl+C to stop\n", addr)

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
