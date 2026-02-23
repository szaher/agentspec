// Package agentspec provides a Go SDK client for the AgentSpec runtime HTTP API.
//
// Usage:
//
//	client := agentspec.NewClient("http://localhost:8080", agentspec.WithAPIKey("my-key"))
//	resp, err := client.Invoke(ctx, "support-bot", "Hello!", nil)
//	fmt.Println(resp.Output)
package agentspec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TokenUsage holds token consumption statistics from an invocation.
type TokenUsage struct {
	Input     int `json:"input"`
	Output    int `json:"output"`
	CacheRead int `json:"cache_read"`
	Total     int `json:"total"`
}

// ToolCall represents a tool call made during an invocation.
type ToolCall struct {
	ID         string                 `json:"id"`
	ToolName   string                 `json:"tool_name"`
	Input      map[string]interface{} `json:"input"`
	Output     interface{}            `json:"output"`
	DurationMs int                    `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
}

// InvokeResponse is the response from an agent invocation.
type InvokeResponse struct {
	Output     string     `json:"output"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Tokens     TokenUsage `json:"tokens"`
	Turns      int        `json:"turns"`
	DurationMs int        `json:"duration_ms"`
	SessionID  string     `json:"session_id,omitempty"`
}

// StreamEvent is a single event from a streaming invocation.
type StreamEvent struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

// AgentInfo holds information about a deployed agent.
type AgentInfo struct {
	Name           string   `json:"name"`
	FQN            string   `json:"fqn"`
	Model          string   `json:"model"`
	Strategy       string   `json:"strategy"`
	Status         string   `json:"status"`
	Skills         []string `json:"skills"`
	ActiveSessions int      `json:"active_sessions"`
}

// SessionInfo holds information about a created session.
type SessionInfo struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	CreatedAt string `json:"created_at"`
}

// PipelineStepResult holds the result of a single pipeline step.
type PipelineStepResult struct {
	Agent      string      `json:"agent"`
	Output     interface{} `json:"output"`
	DurationMs int         `json:"duration_ms"`
	Status     string      `json:"status"`
	Error      string      `json:"error,omitempty"`
}

// PipelineResult holds the result of a pipeline execution.
type PipelineResult struct {
	Pipeline       string                        `json:"pipeline"`
	Status         string                        `json:"status"`
	Steps          map[string]PipelineStepResult  `json:"steps"`
	TotalDurationMs int                           `json:"total_duration_ms"`
	Tokens         TokenUsage                    `json:"tokens"`
}

// HealthResponse is the response from the health check endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Agents  int    `json:"agents"`
	Version string `json:"version"`
}

// APIError represents an error response from the AgentSpec runtime API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	ErrorCode  string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.ErrorCode, e.Message)
}

// InvokeOptions holds optional parameters for an invocation.
type InvokeOptions struct {
	Variables map[string]string
	SessionID string
}

// Option configures the Client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// Client is the AgentSpec runtime API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new AgentSpec client.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			apiErr.ErrorCode = "unknown"
			apiErr.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body, result interface{}) error {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// Health checks the runtime health.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var result HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/healthz", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAgents returns all deployed agents.
func (c *Client) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	var result struct {
		Agents []AgentInfo `json:"agents"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/agents", nil, &result); err != nil {
		return nil, err
	}
	return result.Agents, nil
}

// Invoke invokes an agent and waits for the complete response.
func (c *Client) Invoke(ctx context.Context, agentName, message string, opts *InvokeOptions) (*InvokeResponse, error) {
	body := map[string]interface{}{"message": message}
	if opts != nil {
		if opts.Variables != nil {
			body["variables"] = opts.Variables
		}
		if opts.SessionID != "" {
			body["session_id"] = opts.SessionID
		}
	}

	var result InvokeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/"+agentName+"/invoke", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StreamCallback is called with each streaming event.
type StreamCallback func(event StreamEvent) error

// Stream invokes an agent with streaming and calls the callback for each SSE event.
func (c *Client) Stream(ctx context.Context, agentName, message string, opts *InvokeOptions, callback StreamCallback) error {
	body := map[string]interface{}{"message": message}
	if opts != nil {
		if opts.Variables != nil {
			body["variables"] = opts.Variables
		}
		if opts.SessionID != "" {
			body["session_id"] = opts.SessionID
		}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/v1/agents/"+agentName+"/stream", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	eventType := ""

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = line[7:]
		} else if strings.HasPrefix(line, "data: ") {
			dataStr := line[6:]
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
				data = map[string]interface{}{"raw": dataStr}
			}

			event := StreamEvent{Event: eventType, Data: data}
			if err := callback(event); err != nil {
				return err
			}
			if eventType == "done" {
				return nil
			}
			eventType = ""
		}
	}

	return scanner.Err()
}

// CreateSession creates a new conversation session.
func (c *Client) CreateSession(ctx context.Context, agentName string, metadata map[string]string) (*SessionInfo, error) {
	body := map[string]interface{}{}
	if metadata != nil {
		body["metadata"] = metadata
	}

	var result SessionInfo
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/"+agentName+"/sessions", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendMessage sends a message within an existing session.
func (c *Client) SendMessage(ctx context.Context, agentName, sessionID, message string) (*InvokeResponse, error) {
	body := map[string]interface{}{"message": message}

	var result InvokeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/"+agentName+"/sessions/"+sessionID, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteSession deletes a session and releases memory.
func (c *Client) DeleteSession(ctx context.Context, agentName, sessionID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/v1/agents/"+agentName+"/sessions/"+sessionID, nil, nil)
}

// RunPipeline executes a multi-agent pipeline.
func (c *Client) RunPipeline(ctx context.Context, pipelineName string, trigger map[string]interface{}) (*PipelineResult, error) {
	body := map[string]interface{}{"trigger": trigger}

	var result PipelineResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/pipelines/"+pipelineName+"/run", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
