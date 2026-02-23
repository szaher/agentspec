package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/tools"
)

// Server is the runtime HTTP server for agent invocations.
type Server struct {
	config    *RuntimeConfig
	mux       *http.ServeMux
	server    *http.Server
	logger    *slog.Logger
	llmClient llm.Client
	registry  *tools.Registry
	sessions  *session.Manager
	strategy  loop.Strategy
	startTime time.Time
	apiKey    string
}

// ServerOption configures the Server.
type ServerOption func(*Server)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ServerOption {
	return func(s *Server) { s.apiKey = key }
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// NewServer creates a new runtime HTTP server.
func NewServer(config *RuntimeConfig, llmClient llm.Client, registry *tools.Registry, sessions *session.Manager, strategy loop.Strategy, opts ...ServerOption) *Server {
	s := &Server{
		config:    config,
		llmClient: llmClient,
		registry:  registry,
		sessions:  sessions,
		strategy:  strategy,
		logger:    slog.Default(),
		startTime: time.Now(),
	}
	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /v1/agents", s.handleListAgents)
	mux.HandleFunc("POST /v1/agents/{name}/invoke", s.handleInvoke)
	mux.HandleFunc("POST /v1/agents/{name}/stream", s.handleStream)
	mux.HandleFunc("POST /v1/agents/{name}/sessions", s.handleCreateSession)
	mux.HandleFunc("POST /v1/agents/{name}/sessions/{id}", s.handleSessionMessage)
	mux.HandleFunc("DELETE /v1/agents/{name}/sessions/{id}", s.handleDeleteSession)

	s.mux = mux
	return s
}

// Handler returns the HTTP handler for use with httptest or custom servers.
func (s *Server) Handler() http.Handler {
	return s.authMiddleware(s.mux)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.authMiddleware(s.mux),
	}
	s.logger.Info("runtime server starting", "addr", addr, "agents", len(s.config.Agents))
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check doesn't require auth
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		if s.apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				key = auth[7:]
			}
		}

		if key != s.apiKey {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"uptime":  time.Since(s.startTime).String(),
		"agents":  len(s.config.Agents),
		"version": "0.3.0",
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := make([]map[string]interface{}, len(s.config.Agents))
	for i, a := range s.config.Agents {
		sessions, _ := s.sessions.List(r.Context(), a.Name)
		agents[i] = map[string]interface{}{
			"name":            a.Name,
			"fqn":             a.FQN,
			"model":           a.Model,
			"strategy":        a.Strategy,
			"status":          "running",
			"skills":          a.Skills,
			"active_sessions": len(sessions),
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	agent := s.findAgent(agentName)
	if agent == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	var req struct {
		Message   string            `json:"message"`
		Variables map[string]string `json:"variables,omitempty"`
		SessionID string            `json:"session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Load session messages if provided
	var history []llm.Message
	if req.SessionID != "" {
		var err error
		history, err = s.sessions.LoadMessages(r.Context(), req.SessionID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Session %q not found", req.SessionID))
			return
		}
	}

	inv := loop.Invocation{
		AgentName:   agent.Name,
		Model:       agent.Model,
		System:      agent.System,
		Input:       req.Message,
		Messages:    history,
		Variables:   req.Variables,
		MaxTurns:    agent.MaxTurns,
		MaxTokens:   4096,
		TokenBudget: agent.TokenBudget,
		Temperature: agent.Temperature,
	}

	resp, err := s.strategy.Execute(r.Context(), inv, s.llmClient, s.registry, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Save messages to session
	if req.SessionID != "" {
		msgs := []llm.Message{
			{Role: llm.RoleUser, Content: req.Message},
			{Role: llm.RoleAssistant, Content: resp.Output},
		}
		_ = s.sessions.SaveMessages(r.Context(), req.SessionID, msgs)
	}

	toolCalls := make([]map[string]interface{}, len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		toolCalls[i] = map[string]interface{}{
			"id":          tc.ID,
			"tool_name":   tc.ToolName,
			"input":       tc.Input,
			"output":      tc.Output,
			"duration_ms": tc.Duration.Milliseconds(),
		}
		if tc.Error != "" {
			toolCalls[i]["error"] = tc.Error
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"output":     resp.Output,
		"tool_calls": toolCalls,
		"tokens": map[string]interface{}{
			"input":      resp.Tokens.InputTokens,
			"output":     resp.Tokens.OutputTokens,
			"cache_read": resp.Tokens.CacheRead,
			"total":      resp.Tokens.Total(),
		},
		"turns":       resp.Turns,
		"duration_ms": resp.Duration.Milliseconds(),
		"session_id":  req.SessionID,
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	agent := s.findAgent(agentName)
	if agent == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	var req struct {
		Message   string            `json:"message"`
		Variables map[string]string `json:"variables,omitempty"`
		SessionID string            `json:"session_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported")
		return
	}

	inv := loop.Invocation{
		AgentName:   agent.Name,
		Model:       agent.Model,
		System:      agent.System,
		Input:       req.Message,
		MaxTurns:    agent.MaxTurns,
		MaxTokens:   4096,
		TokenBudget: agent.TokenBudget,
		Temperature: agent.Temperature,
		Stream:      true,
	}

	onEvent := func(event llm.StreamEvent) {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
		flusher.Flush()
	}

	resp, err := s.strategy.Execute(r.Context(), inv, s.llmClient, s.registry, onEvent)
	if err != nil {
		errData, _ := json.Marshal(map[string]string{"error": "internal_error", "message": err.Error()})
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errData))
		flusher.Flush()
		return
	}

	doneData, _ := json.Marshal(map[string]interface{}{
		"tokens": map[string]interface{}{
			"input":  resp.Tokens.InputTokens,
			"output": resp.Tokens.OutputTokens,
			"total":  resp.Tokens.Total(),
		},
		"turns":       resp.Turns,
		"duration_ms": resp.Duration.Milliseconds(),
	})
	_, _ = fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(doneData))
	flusher.Flush()
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	if s.findAgent(agentName) == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	var req struct {
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	sess, err := s.sessions.Create(r.Context(), agentName, req.Metadata)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"session_id": sess.ID,
		"agent":      agentName,
		"created_at": sess.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleSessionMessage(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	sessionID := r.PathValue("id")

	agent := s.findAgent(agentName)
	if agent == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	history, err := s.sessions.LoadMessages(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Session %q not found", sessionID))
		return
	}

	inv := loop.Invocation{
		AgentName:   agent.Name,
		Model:       agent.Model,
		System:      agent.System,
		Input:       req.Message,
		Messages:    history,
		MaxTurns:    agent.MaxTurns,
		MaxTokens:   4096,
		TokenBudget: agent.TokenBudget,
		Temperature: agent.Temperature,
	}

	resp, err := s.strategy.Execute(r.Context(), inv, s.llmClient, s.registry, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: req.Message},
		{Role: llm.RoleAssistant, Content: resp.Output},
	}
	_ = s.sessions.SaveMessages(r.Context(), sessionID, msgs)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"output":      resp.Output,
		"turns":       resp.Turns,
		"duration_ms": resp.Duration.Milliseconds(),
		"session_id":  sessionID,
	})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if err := s.sessions.Close(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) findAgent(name string) *AgentConfig {
	for i := range s.config.Agents {
		if s.config.Agents[i].Name == name {
			return &s.config.Agents[i]
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
