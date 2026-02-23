// Package generator implements the SDK code generation engine.
//
// The generator produces typed client libraries for the AgentSpec runtime
// HTTP API in Python, TypeScript, and Go. Generated clients support agent
// invocation, streaming, session management, and pipeline execution.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/szaher/designs/agentz/internal/runtime"
)

// Language represents a target SDK language.
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangGo         Language = "go"
)

// Config holds SDK generation configuration.
type Config struct {
	Language Language
	OutDir   string
	// RuntimeConfig is the parsed runtime config from IR.
	// If nil, a generic client is generated without agent-specific types.
	RuntimeConfig *runtime.RuntimeConfig
}

// Generate produces SDK files for the configured language.
func Generate(cfg Config) error {
	switch cfg.Language {
	case LangPython:
		return generatePython(cfg)
	case LangTypeScript:
		return generateTypeScript(cfg)
	case LangGo:
		return generateGo(cfg)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}

// GenerateAll produces SDK files for all supported languages.
func GenerateAll(baseDir string, rc *runtime.RuntimeConfig) error {
	for _, lang := range []Language{LangPython, LangTypeScript, LangGo} {
		outDir := filepath.Join(baseDir, string(lang))
		cfg := Config{
			Language:      lang,
			OutDir:        outDir,
			RuntimeConfig: rc,
		}
		if err := Generate(cfg); err != nil {
			return fmt.Errorf("generate %s SDK: %w", lang, err)
		}
	}
	return nil
}

// templateData holds the data passed to SDK templates.
type templateData struct {
	PackageName string
	Agents      []agentData
	Pipelines   []pipelineData
}

type agentData struct {
	Name       string
	NameTitle  string // PascalCase
	NameConst  string // UPPER_SNAKE_CASE
	Model      string
	Strategy   string
	Skills     []string
	HasSession bool
}

type pipelineData struct {
	Name      string
	NameTitle string
	NameConst string // UPPER_SNAKE_CASE
	Steps     []string
}

func buildTemplateData(cfg Config) templateData {
	data := templateData{PackageName: "agentspec"}
	if cfg.RuntimeConfig == nil {
		return data
	}

	data.PackageName = cfg.RuntimeConfig.PackageName
	for _, a := range cfg.RuntimeConfig.Agents {
		data.Agents = append(data.Agents, agentData{
			Name:       a.Name,
			NameTitle:  toTitle(a.Name),
			NameConst:  toUpperSnake(a.Name),
			Model:      a.Model,
			Strategy:   a.Strategy,
			Skills:     a.Skills,
			HasSession: true,
		})
	}
	for _, p := range cfg.RuntimeConfig.Pipelines {
		pd := pipelineData{
			Name:      p.Name,
			NameTitle: toTitle(p.Name),
			NameConst: toUpperSnake(p.Name),
		}
		for _, s := range p.Steps {
			pd.Steps = append(pd.Steps, s.Name)
		}
		data.Pipelines = append(data.Pipelines, pd)
	}
	return data
}

func writeTemplate(path, name, tmplStr string, data interface{}) error {
	t, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", name, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}

	if err := t.Execute(f, data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func toTitle(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func toUpperSnake(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return strings.ToUpper(s)
}

// --- Python Generator ---

const pythonClientTemplate = `"""Generated AgentSpec SDK client.

Auto-generated from runtime API contract. Do not edit manually.
Package: {{ .PackageName }}
"""

from __future__ import annotations

import json
import urllib.request
import urllib.error
from dataclasses import dataclass, field
from typing import Any, Generator, Optional


@dataclass
class TokenUsage:
    """Token usage statistics."""
    input: int = 0
    output: int = 0
    cache_read: int = 0
    total: int = 0


@dataclass
class ToolCall:
    """A tool call made during an invocation."""
    id: str = ""
    tool_name: str = ""
    input: dict[str, Any] = field(default_factory=dict)
    output: Any = None
    duration_ms: int = 0
    error: str = ""


@dataclass
class InvokeResponse:
    """Response from an agent invocation."""
    output: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    tokens: TokenUsage = field(default_factory=TokenUsage)
    turns: int = 0
    duration_ms: int = 0
    session_id: str = ""


@dataclass
class StreamEvent:
    """A single SSE event from a streaming invocation."""
    event: str = ""
    data: dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentInfo:
    """Information about a deployed agent."""
    name: str = ""
    fqn: str = ""
    model: str = ""
    strategy: str = ""
    status: str = ""
    skills: list[str] = field(default_factory=list)
    active_sessions: int = 0


@dataclass
class SessionInfo:
    """Information about a created session."""
    session_id: str = ""
    agent: str = ""
    created_at: str = ""


@dataclass
class PipelineStepResult:
    """Result of a single pipeline step."""
    agent: str = ""
    output: Any = None
    duration_ms: int = 0
    status: str = ""
    error: str = ""


@dataclass
class PipelineResult:
    """Result of a pipeline execution."""
    pipeline: str = ""
    status: str = ""
    steps: dict[str, PipelineStepResult] = field(default_factory=dict)
    total_duration_ms: int = 0
    tokens: TokenUsage = field(default_factory=TokenUsage)


class AgentSpecError(Exception):
    """Base SDK error."""
    pass


class APIError(AgentSpecError):
    """API error response."""
    def __init__(self, status_code: int, error_code: str, message: str):
        super().__init__(f"API error {status_code} ({error_code}): {message}")
        self.status_code = status_code
        self.error_code = error_code


class AgentSpecClient:
    """Client for the AgentSpec runtime HTTP API."""

    def __init__(self, base_url: str = "http://localhost:8080", api_key: str = "", timeout: int = 120):
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        h = {"Content-Type": "application/json"}
        if self._api_key:
            h["Authorization"] = f"Bearer {self._api_key}"
        return h

    def _request(self, method: str, path: str, body: dict | None = None) -> dict:
        url = f"{self._base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, headers=self._headers(), method=method)
        try:
            with urllib.request.urlopen(req, timeout=self._timeout) as resp:
                if resp.status == 204:
                    return {}
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            try:
                err_body = json.loads(e.read().decode())
                raise APIError(e.code, err_body.get("error", "unknown"), err_body.get("message", str(e)))
            except (json.JSONDecodeError, AgentSpecError):
                raise
            except Exception:
                raise APIError(e.code, "unknown", str(e))

    def health(self) -> dict[str, Any]:
        return self._request("GET", "/healthz")

    def list_agents(self) -> list[AgentInfo]:
        resp = self._request("GET", "/v1/agents")
        return [AgentInfo(**a) for a in resp.get("agents", [])]

    def invoke(self, agent_name: str, message: str, variables: dict[str, str] | None = None, session_id: str = "") -> InvokeResponse:
        body: dict[str, Any] = {"message": message}
        if variables:
            body["variables"] = variables
        if session_id:
            body["session_id"] = session_id
        resp = self._request("POST", f"/v1/agents/{agent_name}/invoke", body)
        tokens_data = resp.get("tokens", {})
        tool_calls = [ToolCall(**tc) for tc in resp.get("tool_calls", [])]
        return InvokeResponse(
            output=resp.get("output", ""),
            tool_calls=tool_calls,
            tokens=TokenUsage(**tokens_data),
            turns=resp.get("turns", 0),
            duration_ms=resp.get("duration_ms", 0),
            session_id=resp.get("session_id", ""),
        )

    def stream(self, agent_name: str, message: str, variables: dict[str, str] | None = None) -> Generator[StreamEvent, None, None]:
        body: dict[str, Any] = {"message": message}
        if variables:
            body["variables"] = variables
        url = f"{self._base_url}/v1/agents/{agent_name}/stream"
        data = json.dumps(body).encode()
        req = urllib.request.Request(url, data=data, headers=self._headers(), method="POST")
        resp = urllib.request.urlopen(req, timeout=self._timeout)
        event_type = ""
        for raw_line in resp:
            line = raw_line.decode("utf-8").rstrip("\n\r")
            if line.startswith("event: "):
                event_type = line[7:]
            elif line.startswith("data: "):
                try:
                    data_parsed = json.loads(line[6:])
                except json.JSONDecodeError:
                    data_parsed = {"raw": line[6:]}
                yield StreamEvent(event=event_type, data=data_parsed)
                event_type = ""

    def create_session(self, agent_name: str, metadata: dict[str, str] | None = None) -> SessionInfo:
        body: dict[str, Any] = {}
        if metadata:
            body["metadata"] = metadata
        resp = self._request("POST", f"/v1/agents/{agent_name}/sessions", body)
        return SessionInfo(**resp)

    def send_message(self, agent_name: str, session_id: str, message: str) -> InvokeResponse:
        resp = self._request("POST", f"/v1/agents/{agent_name}/sessions/{session_id}", {"message": message})
        return InvokeResponse(output=resp.get("output", ""), turns=resp.get("turns", 0), duration_ms=resp.get("duration_ms", 0), session_id=resp.get("session_id", ""))

    def delete_session(self, agent_name: str, session_id: str) -> None:
        self._request("DELETE", f"/v1/agents/{agent_name}/sessions/{session_id}")

    def run_pipeline(self, pipeline_name: str, trigger: dict[str, Any]) -> PipelineResult:
        resp = self._request("POST", f"/v1/pipelines/{pipeline_name}/run", {"trigger": trigger})
        steps = {}
        for name, sd in resp.get("steps", {}).items():
            steps[name] = PipelineStepResult(**sd)
        return PipelineResult(pipeline=resp.get("pipeline", ""), status=resp.get("status", ""), steps=steps)
{{ range .Agents }}

# Agent constant: {{ .Name }}
AGENT_{{ .NameConst }} = "{{ .Name }}"
{{ end }}{{ range .Pipelines }}
# Pipeline constant: {{ .Name }}
PIPELINE_{{ .NameConst }} = "{{ .Name }}"
{{ end }}`

func generatePython(cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	data := buildTemplateData(cfg)

	// Write client.py from template
	clientPath := filepath.Join(cfg.OutDir, "client.py")
	if err := writeTemplate(clientPath, "python-client", pythonClientTemplate, data); err != nil {
		return err
	}

	// Write __init__.py
	initContent := `"""AgentSpec SDK - Generated from runtime API contract."""
from .client import (
    AgentSpecClient,
    AgentSpecError,
    APIError,
    AgentInfo,
    InvokeResponse,
    PipelineResult,
    PipelineStepResult,
    SessionInfo,
    StreamEvent,
    TokenUsage,
    ToolCall,
)

__all__ = [
    "AgentSpecClient",
    "AgentSpecError",
    "APIError",
    "AgentInfo",
    "InvokeResponse",
    "PipelineResult",
    "PipelineStepResult",
    "SessionInfo",
    "StreamEvent",
    "TokenUsage",
    "ToolCall",
]

__version__ = "0.1.0"
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "__init__.py"), []byte(initContent), 0644); err != nil {
		return err
	}

	// Write pyproject.toml
	pyprojectContent := `[build-system]
requires = ["setuptools>=68.0", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "agentspec"
version = "0.1.0"
description = "Python SDK for the AgentSpec runtime API"
requires-python = ">=3.10"
license = {text = "MIT"}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		return err
	}

	return nil
}

// --- TypeScript Generator ---

const tsClientTemplate = `// Generated AgentSpec SDK client
// Auto-generated from runtime API contract. Do not edit manually.
// Package: {{ .PackageName }}

export interface TokenUsage {
  input: number;
  output: number;
  cache_read: number;
  total: number;
}

export interface ToolCall {
  id: string;
  tool_name: string;
  input: Record<string, unknown>;
  output: unknown;
  duration_ms: number;
  error?: string;
}

export interface InvokeResponse {
  output: string;
  tool_calls: ToolCall[];
  tokens: TokenUsage;
  turns: number;
  duration_ms: number;
  session_id: string;
}

export interface StreamEvent {
  event: string;
  data: Record<string, unknown>;
}

export interface AgentInfo {
  name: string;
  fqn: string;
  model: string;
  strategy: string;
  status: string;
  skills: string[];
  active_sessions: number;
}

export interface SessionInfo {
  session_id: string;
  agent: string;
  created_at: string;
}

export interface PipelineStepResult {
  agent: string;
  output: unknown;
  duration_ms: number;
  status: string;
  error?: string;
}

export interface PipelineResult {
  pipeline: string;
  status: string;
  steps: Record<string, PipelineStepResult>;
  total_duration_ms: number;
  tokens: TokenUsage;
}

export class AgentSpecError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AgentSpecError";
  }
}

export class APIError extends AgentSpecError {
  public readonly statusCode: number;
  public readonly errorCode: string;

  constructor(statusCode: number, errorCode: string, message: string) {
    super(` + "`" + `API error ${statusCode} (${errorCode}): ${message}` + "`" + `);
    this.name = "APIError";
    this.statusCode = statusCode;
    this.errorCode = errorCode;
  }
}

export class AgentSpecClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly timeout: number;

  constructor(options: { baseUrl?: string; apiKey?: string; timeout?: number } = {}) {
    this.baseUrl = (options.baseUrl || "http://localhost:8080").replace(/\/$/, "");
    this.apiKey = options.apiKey || "";
    this.timeout = options.timeout || 120000;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { "Content-Type": "application/json" };
    if (this.apiKey) h["Authorization"] = ` + "`" + `Bearer ${this.apiKey}` + "`" + `;
    return h;
  }

  private async request<T>(method: string, path: string, body?: Record<string, unknown>): Promise<T> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);
    try {
      const resp = await fetch(` + "`" + `${this.baseUrl}${path}` + "`" + `, {
        method, headers: this.headers(),
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });
      if (resp.status === 204) return {} as T;
      const data = await resp.json();
      if (!resp.ok) throw new APIError(resp.status, data.error || "unknown", data.message || "failed");
      return data as T;
    } finally { clearTimeout(timer); }
  }

  async health(): Promise<Record<string, unknown>> { return this.request("GET", "/healthz"); }

  async listAgents(): Promise<AgentInfo[]> {
    const resp = await this.request<{ agents: AgentInfo[] }>("GET", "/v1/agents");
    return resp.agents || [];
  }

  async invoke(agentName: string, message: string, options?: { variables?: Record<string, string>; sessionId?: string }): Promise<InvokeResponse> {
    const body: Record<string, unknown> = { message };
    if (options?.variables) body.variables = options.variables;
    if (options?.sessionId) body.session_id = options.sessionId;
    return this.request("POST", ` + "`" + `/v1/agents/${agentName}/invoke` + "`" + `, body);
  }

  async createSession(agentName: string, metadata?: Record<string, string>): Promise<SessionInfo> {
    return this.request("POST", ` + "`" + `/v1/agents/${agentName}/sessions` + "`" + `, metadata ? { metadata } : {});
  }

  async sendMessage(agentName: string, sessionId: string, message: string): Promise<InvokeResponse> {
    return this.request("POST", ` + "`" + `/v1/agents/${agentName}/sessions/${sessionId}` + "`" + `, { message });
  }

  async deleteSession(agentName: string, sessionId: string): Promise<void> {
    await this.request("DELETE", ` + "`" + `/v1/agents/${agentName}/sessions/${sessionId}` + "`" + `);
  }

  async runPipeline(pipelineName: string, trigger: Record<string, unknown>): Promise<PipelineResult> {
    return this.request("POST", ` + "`" + `/v1/pipelines/${pipelineName}/run` + "`" + `, { trigger });
  }
}
{{ range .Agents }}
/** Agent constant: {{ .Name }} */
export const AGENT_{{ .NameConst }} = "{{ .Name }}";
{{ end }}{{ range .Pipelines }}
/** Pipeline constant: {{ .Name }} */
export const PIPELINE_{{ .NameConst }} = "{{ .Name }}";
{{ end }}`

func generateTypeScript(cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	data := buildTemplateData(cfg)

	// Write client.ts from template
	clientPath := filepath.Join(cfg.OutDir, "index.ts")
	if err := writeTemplate(clientPath, "ts-client", tsClientTemplate, data); err != nil {
		return err
	}

	// Write package.json
	pkgContent := `{
  "name": "@agentspec/sdk",
  "version": "0.1.0",
  "description": "TypeScript SDK for the AgentSpec runtime API",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": { "build": "tsc" },
  "license": "MIT",
  "devDependencies": { "typescript": "^5.4.0" }
}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "package.json"), []byte(pkgContent), 0644); err != nil {
		return err
	}

	// Write tsconfig.json
	tsconfigContent := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "outDir": "dist",
    "rootDir": ".",
    "declaration": true,
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["*.ts"]
}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "tsconfig.json"), []byte(tsconfigContent), 0644); err != nil {
		return err
	}

	return nil
}

// --- Go Generator ---

const goClientTemplate = `// Package agentspec provides a generated Go SDK for the AgentSpec runtime API.
//
// Auto-generated from runtime API contract. Do not edit manually.
// Package: {{ .PackageName }}
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

// TokenUsage holds token consumption statistics.
type TokenUsage struct {
	Input     int ` + "`" + `json:"input"` + "`" + `
	Output    int ` + "`" + `json:"output"` + "`" + `
	CacheRead int ` + "`" + `json:"cache_read"` + "`" + `
	Total     int ` + "`" + `json:"total"` + "`" + `
}

// ToolCall represents a tool call made during an invocation.
type ToolCall struct {
	ID         string                 ` + "`" + `json:"id"` + "`" + `
	ToolName   string                 ` + "`" + `json:"tool_name"` + "`" + `
	Input      map[string]interface{} ` + "`" + `json:"input"` + "`" + `
	Output     interface{}            ` + "`" + `json:"output"` + "`" + `
	DurationMs int                    ` + "`" + `json:"duration_ms"` + "`" + `
	Error      string                 ` + "`" + `json:"error,omitempty"` + "`" + `
}

// InvokeResponse is the response from an agent invocation.
type InvokeResponse struct {
	Output     string     ` + "`" + `json:"output"` + "`" + `
	ToolCalls  []ToolCall ` + "`" + `json:"tool_calls,omitempty"` + "`" + `
	Tokens     TokenUsage ` + "`" + `json:"tokens"` + "`" + `
	Turns      int        ` + "`" + `json:"turns"` + "`" + `
	DurationMs int        ` + "`" + `json:"duration_ms"` + "`" + `
	SessionID  string     ` + "`" + `json:"session_id,omitempty"` + "`" + `
}

// StreamEvent is a single SSE event.
type StreamEvent struct {
	Event string                 ` + "`" + `json:"event"` + "`" + `
	Data  map[string]interface{} ` + "`" + `json:"data"` + "`" + `
}

// AgentInfo holds deployed agent information.
type AgentInfo struct {
	Name           string   ` + "`" + `json:"name"` + "`" + `
	FQN            string   ` + "`" + `json:"fqn"` + "`" + `
	Model          string   ` + "`" + `json:"model"` + "`" + `
	Strategy       string   ` + "`" + `json:"strategy"` + "`" + `
	Status         string   ` + "`" + `json:"status"` + "`" + `
	Skills         []string ` + "`" + `json:"skills"` + "`" + `
	ActiveSessions int      ` + "`" + `json:"active_sessions"` + "`" + `
}

// SessionInfo holds session information.
type SessionInfo struct {
	SessionID string ` + "`" + `json:"session_id"` + "`" + `
	Agent     string ` + "`" + `json:"agent"` + "`" + `
	CreatedAt string ` + "`" + `json:"created_at"` + "`" + `
}

// PipelineStepResult holds a pipeline step result.
type PipelineStepResult struct {
	Agent      string      ` + "`" + `json:"agent"` + "`" + `
	Output     interface{} ` + "`" + `json:"output"` + "`" + `
	DurationMs int         ` + "`" + `json:"duration_ms"` + "`" + `
	Status     string      ` + "`" + `json:"status"` + "`" + `
	Error      string      ` + "`" + `json:"error,omitempty"` + "`" + `
}

// PipelineResult holds a pipeline execution result.
type PipelineResult struct {
	Pipeline        string                        ` + "`" + `json:"pipeline"` + "`" + `
	Status          string                        ` + "`" + `json:"status"` + "`" + `
	Steps           map[string]PipelineStepResult  ` + "`" + `json:"steps"` + "`" + `
	TotalDurationMs int                           ` + "`" + `json:"total_duration_ms"` + "`" + `
	Tokens          TokenUsage                    ` + "`" + `json:"tokens"` + "`" + `
}

// APIError represents an API error response.
type APIError struct {
	StatusCode int    ` + "`" + `json:"status_code"` + "`" + `
	ErrorCode  string ` + "`" + `json:"error"` + "`" + `
	Message    string ` + "`" + `json:"message"` + "`" + `
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.ErrorCode, e.Message)
}

// Option configures the Client.
type Option func(*Client)

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.httpClient = hc } }

// Client is the AgentSpec runtime API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new client.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: &http.Client{Timeout: 120 * time.Second}}
	for _, o := range opts { o(c) }
	return c
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil { return nil, err }
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" { req.Header.Set("Authorization", "Bearer "+c.apiKey) }
	resp, err := c.httpClient.Do(req)
	if err != nil { return nil, err }
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var ae APIError
		ae.StatusCode = resp.StatusCode
		_ = json.NewDecoder(resp.Body).Decode(&ae)
		return nil, &ae
	}
	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body, result interface{}) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode == 204 { return nil }
	return json.NewDecoder(resp.Body).Decode(result)
}

// Invoke invokes an agent.
func (c *Client) Invoke(ctx context.Context, agentName, message string, vars map[string]string) (*InvokeResponse, error) {
	body := map[string]interface{}{"message": message}
	if vars != nil { body["variables"] = vars }
	var result InvokeResponse
	return &result, c.doJSON(ctx, "POST", "/v1/agents/"+agentName+"/invoke", body, &result)
}

// Stream invokes an agent with streaming.
func (c *Client) Stream(ctx context.Context, agentName, message string, callback func(StreamEvent) error) error {
	resp, err := c.do(ctx, "POST", "/v1/agents/"+agentName+"/stream", map[string]interface{}{"message": message})
	if err != nil { return err }
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	eventType := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") { eventType = line[7:] }
		if strings.HasPrefix(line, "data: ") {
			var data map[string]interface{}
			_ = json.Unmarshal([]byte(line[6:]), &data)
			if err := callback(StreamEvent{Event: eventType, Data: data}); err != nil { return err }
			if eventType == "done" { return nil }
			eventType = ""
		}
	}
	return scanner.Err()
}

// ListAgents returns all deployed agents.
func (c *Client) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	var result struct{ Agents []AgentInfo ` + "`" + `json:"agents"` + "`" + ` }
	return result.Agents, c.doJSON(ctx, "GET", "/v1/agents", nil, &result)
}

// CreateSession creates a session.
func (c *Client) CreateSession(ctx context.Context, agentName string) (*SessionInfo, error) {
	var result SessionInfo
	return &result, c.doJSON(ctx, "POST", "/v1/agents/"+agentName+"/sessions", map[string]interface{}{}, &result)
}

// SendMessage sends a session message.
func (c *Client) SendMessage(ctx context.Context, agentName, sessionID, message string) (*InvokeResponse, error) {
	var result InvokeResponse
	return &result, c.doJSON(ctx, "POST", "/v1/agents/"+agentName+"/sessions/"+sessionID, map[string]interface{}{"message": message}, &result)
}

// DeleteSession deletes a session.
func (c *Client) DeleteSession(ctx context.Context, agentName, sessionID string) error {
	return c.doJSON(ctx, "DELETE", "/v1/agents/"+agentName+"/sessions/"+sessionID, nil, nil)
}

// RunPipeline executes a pipeline.
func (c *Client) RunPipeline(ctx context.Context, pipelineName string, trigger map[string]interface{}) (*PipelineResult, error) {
	var result PipelineResult
	return &result, c.doJSON(ctx, "POST", "/v1/pipelines/"+pipelineName+"/run", map[string]interface{}{"trigger": trigger}, &result)
}
{{ range .Agents }}
// Agent{{ .NameTitle }} is the agent name constant for "{{ .Name }}".
const Agent{{ .NameTitle }} = "{{ .Name }}"
{{ end }}{{ range .Pipelines }}
// Pipeline{{ .NameTitle }} is the pipeline name constant for "{{ .Name }}".
const Pipeline{{ .NameTitle }} = "{{ .Name }}"
{{ end }}`

func generateGo(cfg Config) error {
	outDir := filepath.Join(cfg.OutDir, "agentspec")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	data := buildTemplateData(cfg)

	// Write client.go from template
	clientPath := filepath.Join(outDir, "client.go")
	if err := writeTemplate(clientPath, "go-client", goClientTemplate, data); err != nil {
		return err
	}

	// Write go.mod
	goModContent := "module github.com/szaher/designs/agentz/sdk/go\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return err
	}

	return nil
}
