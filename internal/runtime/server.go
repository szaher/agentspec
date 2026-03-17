package runtime

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/szaher/agentspec/internal/auth"
	"github.com/szaher/agentspec/internal/controlflow"
	"github.com/szaher/agentspec/internal/cost"
	"github.com/szaher/agentspec/internal/eviction"
	"github.com/szaher/agentspec/internal/frontend"
	"github.com/szaher/agentspec/internal/llm"
	"github.com/szaher/agentspec/internal/loop"
	"github.com/szaher/agentspec/internal/session"
	"github.com/szaher/agentspec/internal/telemetry"
	"github.com/szaher/agentspec/internal/tools"
)

// Server is the runtime HTTP server for agent invocations.
type Server struct {
	config      *RuntimeConfig
	mux         *http.ServeMux
	server      *http.Server
	logger      *slog.Logger
	llmClient   llm.Client
	registry    *tools.Registry
	sessions    *session.Manager
	strategy    loop.Strategy
	startTime   time.Time
	apiKey      string
	noAuth      bool
	corsOrigins []string
	metrics     *telemetry.Metrics
	rateLimiter *auth.RateLimiter
	enableUI    bool

	agentIndex    map[string]*AgentConfig
	pipelineIndex map[string]*PipelineConfig
	userStore     *auth.UserStore
	auditLogger   *auth.AuditLogger
	costTracker   *cost.CostTracker

	tlsCertFile string
	tlsKeyFile  string
	tlsCertMu   sync.RWMutex
	tlsCert     *tls.Certificate
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

// WithMetrics sets the metrics collector.
func WithMetrics(m *telemetry.Metrics) ServerOption {
	return func(s *Server) { s.metrics = m }
}

// WithRateLimit sets the per-agent rate limit (requests per second and burst).
func WithRateLimit(rate float64, burst int) ServerOption {
	return func(s *Server) {
		s.rateLimiter = auth.NewRateLimiter(auth.RateLimitConfig{
			RequestsPerSecond: rate,
			Burst:             burst,
		}, eviction.DefaultPolicy())
	}
}

// WithNoAuth explicitly allows unauthenticated access.
func WithNoAuth(noAuth bool) ServerOption {
	return func(s *Server) { s.noAuth = noAuth }
}

// WithCORSOrigins sets the allowed CORS origins.
func WithCORSOrigins(origins []string) ServerOption {
	return func(s *Server) { s.corsOrigins = origins }
}

// WithUI enables the built-in web frontend.
func WithUI(enable bool) ServerOption {
	return func(s *Server) { s.enableUI = enable }
}

// WithTLS configures TLS certificate and key files.
func WithTLS(certFile, keyFile string) ServerOption {
	return func(s *Server) {
		s.tlsCertFile = certFile
		s.tlsKeyFile = keyFile
	}
}

// WithUserStore sets the multi-user authentication store.
func WithUserStore(store *auth.UserStore) ServerOption {
	return func(s *Server) { s.userStore = store }
}

// WithAuditLogger sets the audit logger for invocation tracking.
func WithAuditLogger(logger *auth.AuditLogger) ServerOption {
	return func(s *Server) { s.auditLogger = logger }
}

// WithCostTracker sets the cost tracker for budget enforcement.
func WithCostTracker(ct *cost.CostTracker) ServerOption {
	return func(s *Server) { s.costTracker = ct }
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

	// Build lookup indexes for O(1) agent and pipeline access
	s.agentIndex = make(map[string]*AgentConfig, len(config.Agents))
	for i := range config.Agents {
		s.agentIndex[config.Agents[i].Name] = &config.Agents[i]
	}
	s.pipelineIndex = make(map[string]*PipelineConfig, len(config.Pipelines))
	for i := range config.Pipelines {
		s.pipelineIndex[config.Pipelines[i].Name] = &config.Pipelines[i]
	}

	if s.metrics == nil {
		s.metrics = telemetry.NewMetrics()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /v1/agents", s.handleListAgents)
	mux.HandleFunc("POST /v1/agents/{name}/invoke", limitBody(s.rateLimitMiddleware(s.handleInvoke)))
	mux.HandleFunc("POST /v1/agents/{name}/stream", limitBody(s.rateLimitMiddleware(s.handleStream)))
	mux.HandleFunc("POST /v1/agents/{name}/sessions", limitBody(s.handleCreateSession))
	mux.HandleFunc("POST /v1/agents/{name}/sessions/{id}", limitBody(s.handleSessionMessage))
	mux.HandleFunc("DELETE /v1/agents/{name}/sessions/{id}", s.handleDeleteSession)
	mux.HandleFunc("POST /v1/pipelines/{name}/run", limitBody(s.handlePipelineRun))
	mux.Handle("GET /v1/metrics", s.metrics.Handler())

	// Mount built-in frontend when enabled
	if s.enableUI {
		frontend.Mount(mux, frontend.NewHandler("/"))
	}

	s.mux = mux
	return s
}

// Handler returns the HTTP handler for use with httptest or custom servers.
func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.authMiddleware(telemetry.CorrelationMiddleware(s.mux)))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && len(s.corsOrigins) > 0 {
			for _, allowed := range s.corsOrigins {
				if allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
					w.Header().Set("Access-Control-Max-Age", "86400")
					break
				}
			}
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ListenAndServe starts the HTTP server with production-ready timeouts.
// When TLS cert/key are configured, serves HTTPS with certificate hot-reload.
func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.corsMiddleware(s.authMiddleware(telemetry.CorrelationMiddleware(s.mux))),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if s.tlsCertFile != "" && s.tlsKeyFile != "" {
		// Validate cert/key at startup
		cert, err := tls.LoadX509KeyPair(s.tlsCertFile, s.tlsKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		s.tlsCertMu.Lock()
		s.tlsCert = &cert
		s.tlsCertMu.Unlock()

		// Configure TLS with hot-reload via GetCertificate callback
		s.server.TLSConfig = &tls.Config{
			GetCertificate: s.getCertificate,
			MinVersion:     tls.VersionTLS12,
		}

		s.logger.Info("runtime server starting with TLS", "addr", addr, "cert", s.tlsCertFile, "agents", len(s.config.Agents))
		return s.server.ListenAndServeTLS("", "")
	}

	if s.tlsCertFile != "" && s.tlsKeyFile == "" {
		return fmt.Errorf("--tls-cert provided but --tls-key is missing")
	}
	if s.tlsKeyFile != "" && s.tlsCertFile == "" {
		return fmt.Errorf("--tls-key provided but --tls-cert is missing")
	}

	s.logger.Info("runtime server starting", "addr", addr, "agents", len(s.config.Agents))
	s.logger.Warn("TLS is not configured — serving plain HTTP")
	return s.server.ListenAndServe()
}

// getCertificate provides the TLS certificate for each handshake, supporting hot-reload.
func (s *Server) getCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.tlsCertMu.RLock()
	cert := s.tlsCert
	s.tlsCertMu.RUnlock()
	if cert == nil {
		return nil, fmt.Errorf("no TLS certificate loaded")
	}
	return cert, nil
}

// ReloadTLSCertificate reloads the TLS certificate from disk.
// Called by the file watcher when cert/key files change.
func (s *Server) ReloadTLSCertificate() error {
	if s.tlsCertFile == "" || s.tlsKeyFile == "" {
		return nil
	}
	cert, err := tls.LoadX509KeyPair(s.tlsCertFile, s.tlsKeyFile)
	if err != nil {
		s.logger.Error("failed to reload TLS certificate", "error", err)
		return err
	}
	s.tlsCertMu.Lock()
	s.tlsCert = &cert
	s.tlsCertMu.Unlock()
	s.logger.Info("TLS certificate reloaded", "cert", s.tlsCertFile)
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

type contextKey string

const userContextKey contextKey = "user"

// UserFromContext extracts the authenticated user from the request context.
func UserFromContext(ctx context.Context) *auth.User {
	u, _ := ctx.Value(userContextKey).(*auth.User)
	return u
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check doesn't require auth
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		// Metrics endpoint doesn't require auth
		if r.URL.Path == "/v1/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Frontend static assets don't require auth
		if s.enableUI && !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key from header
		key := r.Header.Get("X-API-Key")
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = authHeader[7:]
			}
		}

		// Multi-user auth mode: resolve key to user identity
		if s.userStore != nil {
			user, ok := s.userStore.Resolve(key)
			if !ok {
				if s.noAuth && key == "" {
					next.ServeHTTP(w, r)
					return
				}
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid API key")
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Single-key auth mode (backward compatible)
		if s.apiKey == "" {
			if s.noAuth {
				next.ServeHTTP(w, r)
				return
			}
			writeError(w, http.StatusUnauthorized, "unauthorized", "No API key configured. Use --no-auth to allow unauthenticated access.")
			return
		}

		if !auth.ValidateKey(key, s.apiKey) {
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
	agent := s.agentIndex[agentName]
	if agent == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	// Check per-user agent access
	user := UserFromContext(r.Context())
	if user != nil && !user.IsAuthorized(agentName) {
		if s.auditLogger != nil {
			s.auditLogger.Log(auth.AuditEntry{
				User:          user.Name,
				Agent:         agentName,
				Action:        "invoke",
				Status:        "denied",
				CorrelationID: telemetry.CorrelationID(r.Context()),
			})
		}
		writeError(w, http.StatusForbidden, "forbidden", fmt.Sprintf("User %q is not authorized to access agent %q", user.Name, agentName))
		return
	}

	logger := telemetry.RequestLogger(s.logger, r.Context(), agentName)
	_ = logger // available for future logging within this handler

	// Budget check before invocation
	if s.costTracker != nil {
		if err := s.costTracker.CheckBudget(agentName); err != nil {
			w.Header().Set("Retry-After", "3600")
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":   "budget_exceeded",
				"message": err.Error(),
				"agent":   agentName,
			})
			return
		}
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

	// If agent has on_input control flow, execute it instead of default LLM flow
	if len(agent.OnInput) > 0 {
		s.handleControlFlowInvoke(w, r, agent, req.Message)
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

	start := time.Now()
	resp, err := s.strategy.Execute(r.Context(), inv, s.llmClient, s.registry, nil)
	if err != nil {
		s.metrics.RecordInvocation(agentName, agent.Model, "failed", time.Since(start), 0, 0)
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Record metrics
	s.metrics.RecordInvocation(agentName, agent.Model, "completed", time.Since(start), resp.Tokens.InputTokens, resp.Tokens.OutputTokens)
	for _, tc := range resp.ToolCalls {
		status := "success"
		if tc.Error != "" {
			status = "error"
		}
		s.metrics.RecordToolCall(agentName, tc.ToolName, status)
	}

	// Record cost and update budget
	if s.costTracker != nil {
		invCost := s.costTracker.RecordUsage(agentName, agent.Model, resp.Tokens.InputTokens, resp.Tokens.OutputTokens)
		s.metrics.RecordCost(agentName, agent.Model, invCost)

		if warn, entry := s.costTracker.CheckWarnings(agentName); warn {
			s.logger.Warn("budget warning: 80% threshold reached",
				"agent", agentName,
				"period", entry.Period,
				"used", entry.UsedDollars,
				"limit", entry.LimitDollars,
			)
		}
	}

	// Save messages to session
	if req.SessionID != "" {
		msgs := []llm.Message{
			{Role: llm.RoleUser, Content: req.Message},
			{Role: llm.RoleAssistant, Content: resp.Output},
		}
		if err := s.sessions.SaveMessages(r.Context(), req.SessionID, msgs); err != nil {
			slog.Error("failed to save session messages", "session_id", req.SessionID, "error", err)
			w.Header().Set("Warning", `199 - "session save failed"`)
		}
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

	// Audit log
	if s.auditLogger != nil {
		userName := ""
		if user != nil {
			userName = user.Name
		}
		s.auditLogger.Log(auth.AuditEntry{
			User:          userName,
			Agent:         agentName,
			SessionID:     req.SessionID,
			Action:        "invoke",
			InputTokens:   resp.Tokens.InputTokens,
			OutputTokens:  resp.Tokens.OutputTokens,
			DurationMs:    resp.Duration.Milliseconds(),
			Status:        "success",
			CorrelationID: telemetry.CorrelationID(r.Context()),
		})
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
	agent := s.agentIndex[agentName]
	if agent == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Agent %q not found", agentName))
		return
	}

	logger := telemetry.RequestLogger(s.logger, r.Context(), agentName)
	_ = logger // available for future logging within this handler

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
	if s.agentIndex[agentName] == nil {
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

	agent := s.agentIndex[agentName]
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
	if err := s.sessions.SaveMessages(r.Context(), sessionID, msgs); err != nil {
		slog.Error("failed to save session messages", "session_id", sessionID, "error", err)
		w.Header().Set("Warning", `199 - "session save failed"`)
	}

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

func (s *Server) handlePipelineRun(w http.ResponseWriter, r *http.Request) {
	pipelineName := r.PathValue("name")
	pConfig := s.pipelineIndex[pipelineName]
	if pConfig == nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("Pipeline %q not found", pipelineName))
		return
	}

	logger := telemetry.RequestLogger(s.logger, r.Context(), pipelineName)
	_ = logger // available for future logging within this handler

	var req struct {
		Trigger map[string]interface{} `json:"trigger"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	triggerInput, _ := json.Marshal(req.Trigger)

	// Build pipeline steps
	var steps []pipelineStep
	for _, step := range pConfig.Steps {
		steps = append(steps, pipelineStep{
			Name:      step.Name,
			AgentRef:  step.AgentRef,
			Input:     step.Input,
			DependsOn: step.DependsOn,
		})
	}

	// Execute pipeline using inline invocation
	result := s.executePipeline(r.Context(), pipelineName, steps, string(triggerInput))

	writeJSON(w, http.StatusOK, result)
}

type pipelineStep struct {
	Name      string
	AgentRef  string
	Input     string
	DependsOn []string
}

func (s *Server) executePipeline(ctx context.Context, name string, steps []pipelineStep, triggerInput string) map[string]interface{} {
	result := map[string]interface{}{
		"pipeline": name,
		"status":   "completed",
		"steps":    map[string]interface{}{},
	}

	stepOutputs := map[string]string{"trigger": triggerInput}
	stepsResult, _ := result["steps"].(map[string]interface{})

	for _, step := range steps {
		input := step.Input
		if input == "" && len(step.DependsOn) > 0 {
			input = stepOutputs[step.DependsOn[0]]
		}
		if input == "" {
			input = triggerInput
		}

		agent := s.agentIndex[step.AgentRef]
		if agent == nil {
			stepsResult[step.Name] = map[string]interface{}{
				"agent":  step.AgentRef,
				"status": "failed",
				"error":  fmt.Sprintf("agent %q not found", step.AgentRef),
			}
			result["status"] = "failed"
			break
		}

		inv := loop.Invocation{
			AgentName:   agent.Name,
			Model:       agent.Model,
			System:      agent.System,
			Input:       input,
			MaxTurns:    agent.MaxTurns,
			MaxTokens:   4096,
			TokenBudget: agent.TokenBudget,
			Temperature: agent.Temperature,
		}

		resp, err := s.strategy.Execute(ctx, inv, s.llmClient, s.registry, nil)
		if err != nil {
			stepsResult[step.Name] = map[string]interface{}{
				"agent":  step.AgentRef,
				"status": "failed",
				"error":  err.Error(),
			}
			result["status"] = "failed"
			break
		}

		stepOutputs[step.Name] = resp.Output
		stepsResult[step.Name] = map[string]interface{}{
			"agent":       step.AgentRef,
			"output":      resp.Output,
			"duration_ms": resp.Duration.Milliseconds(),
			"status":      "completed",
		}
	}

	return result
}

// handleControlFlowInvoke processes an invoke request using the agent's on_input control flow.
func (s *Server) handleControlFlowInvoke(w http.ResponseWriter, r *http.Request, agent *AgentConfig, message string) {
	rc := controlflow.NewRuntimeContext(message, nil, nil)

	// Build skill invoker that uses the LLM or registered tools
	skillInvoker := &serverSkillInvoker{server: s, agent: agent}
	agentDelegator := &serverAgentDelegator{server: s}

	executor := controlflow.NewExecutor(skillInvoker, agentDelegator)

	start := time.Now()
	actions, output, err := executor.ExecuteBlock(r.Context(), agent.OnInput, rc)
	if err != nil {
		s.metrics.RecordInvocation(agent.Name, agent.Model, "failed", time.Since(start), 0, 0)
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	s.metrics.RecordInvocation(agent.Name, agent.Model, "completed", time.Since(start), 0, 0)

	// Build activity trace from actions
	activity := make([]map[string]interface{}, len(actions))
	for i, a := range actions {
		entry := map[string]interface{}{
			"type": a.Type,
		}
		switch a.Type {
		case "use_skill":
			entry["content"] = fmt.Sprintf("Invoked skill: %s", a.SkillName)
		case "delegate":
			entry["content"] = fmt.Sprintf("Delegated to agent: %s", a.AgentName)
		case "respond":
			entry["content"] = fmt.Sprintf("Responded with expression: %s", a.Expression)
		}
		activity[i] = entry
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"output":      output,
		"activity":    activity,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}

// serverSkillInvoker invokes skills through the server's registered tools or LLM.
type serverSkillInvoker struct {
	server *Server
	agent  *AgentConfig
}

func (si *serverSkillInvoker) InvokeSkill(ctx context.Context, skillName string, params map[string]string, input interface{}) (string, error) {
	// Try registered tool first
	if si.server.registry != nil {
		toolInput := make(map[string]interface{})
		for k, v := range params {
			toolInput[k] = v
		}
		if input != nil {
			toolInput["input"] = input
		}
		call := llm.ToolCall{
			ID:    session.GenerateID("cf_"),
			Name:  skillName,
			Input: toolInput,
		}
		result, err := si.server.registry.Execute(ctx, call)
		if err == nil {
			return result, nil
		}
		// If tool not found, fall through to LLM
	}

	// Fall back to LLM invocation with skill context
	inv := loop.Invocation{
		AgentName: si.agent.Name,
		Model:     si.agent.Model,
		System:    si.agent.System,
		Input:     fmt.Sprintf("Use the skill '%s' to process: %v", skillName, input),
		MaxTurns:  si.agent.MaxTurns,
		MaxTokens: 4096,
	}

	resp, err := si.server.strategy.Execute(ctx, inv, si.server.llmClient, si.server.registry, nil)
	if err != nil {
		return "", err
	}
	return resp.Output, nil
}

// serverAgentDelegator delegates to another agent through the server.
type serverAgentDelegator struct {
	server *Server
}

func (ad *serverAgentDelegator) DelegateToAgent(ctx context.Context, agentName string, input interface{}) (string, error) {
	agent := ad.server.agentIndex[agentName]
	if agent == nil {
		return "", fmt.Errorf("agent %q not found for delegation", agentName)
	}

	inv := loop.Invocation{
		AgentName: agent.Name,
		Model:     agent.Model,
		System:    agent.System,
		Input:     fmt.Sprintf("%v", input),
		MaxTurns:  agent.MaxTurns,
		MaxTokens: 4096,
	}

	resp, err := ad.server.strategy.Execute(ctx, inv, ad.server.llmClient, ad.server.registry, nil)
	if err != nil {
		return "", err
	}
	return resp.Output, nil
}

func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.rateLimiter == nil {
			next(w, r)
			return
		}

		agentName := r.PathValue("name")
		if agentName == "" {
			next(w, r)
			return
		}

		if !s.rateLimiter.Allow(agentName) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "Per-agent rate limit exceeded")
			return
		}

		next(w, r)
	}
}

const maxRequestBodySize = 10 * 1024 * 1024 // 10MB

// limitBody wraps an http.HandlerFunc to limit the request body size.
func limitBody(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		}
		next(w, r)
	}
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
