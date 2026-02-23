/**
 * AgentSpec SDK client for TypeScript.
 *
 * Provides typed methods for invoking agents, streaming responses,
 * and managing sessions via the AgentSpec runtime HTTP API.
 */

/** Token usage statistics from an invocation. */
export interface TokenUsage {
  input: number;
  output: number;
  cache_read: number;
  total: number;
}

/** A tool call made during an invocation. */
export interface ToolCall {
  id: string;
  tool_name: string;
  input: Record<string, unknown>;
  output: unknown;
  duration_ms: number;
  error?: string;
}

/** Response from an agent invocation. */
export interface InvokeResponse {
  output: string;
  tool_calls: ToolCall[];
  tokens: TokenUsage;
  turns: number;
  duration_ms: number;
  session_id: string;
}

/** A single event from a streaming invocation. */
export interface StreamEvent {
  event: string;
  data: Record<string, unknown>;
}

/** Information about a deployed agent. */
export interface AgentInfo {
  name: string;
  fqn: string;
  model: string;
  strategy: string;
  status: string;
  skills: string[];
  active_sessions: number;
}

/** Information about a created session. */
export interface SessionInfo {
  session_id: string;
  agent: string;
  created_at: string;
}

/** Result of a single pipeline step. */
export interface PipelineStepResult {
  agent: string;
  output: unknown;
  duration_ms: number;
  status: string;
  error?: string;
}

/** Result of a pipeline execution. */
export interface PipelineResult {
  pipeline: string;
  status: string;
  steps: Record<string, PipelineStepResult>;
  total_duration_ms: number;
  tokens: TokenUsage;
}

/** Health check response. */
export interface HealthResponse {
  status: string;
  uptime: string;
  agents: number;
  version: string;
}

/** Base error for AgentSpec SDK. */
export class AgentSpecError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AgentSpecError";
  }
}

/** Error returned by the AgentSpec runtime API. */
export class APIError extends AgentSpecError {
  public readonly statusCode: number;
  public readonly errorCode: string;

  constructor(statusCode: number, errorCode: string, message: string) {
    super(`API error ${statusCode} (${errorCode}): ${message}`);
    this.name = "APIError";
    this.statusCode = statusCode;
    this.errorCode = errorCode;
  }
}

/** Client configuration options. */
export interface ClientOptions {
  baseUrl?: string;
  apiKey?: string;
  timeout?: number;
}

/**
 * Client for the AgentSpec runtime HTTP API.
 *
 * @example
 * ```ts
 * const client = new AgentSpecClient({ baseUrl: "http://localhost:8080" });
 * const response = await client.invoke("support-bot", "Hello!");
 * console.log(response.output);
 * ```
 */
export class AgentSpecClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly timeout: number;

  constructor(options: ClientOptions = {}) {
    this.baseUrl = (options.baseUrl || "http://localhost:8080").replace(
      /\/$/,
      ""
    );
    this.apiKey = options.apiKey || "";
    this.timeout = options.timeout || 120000;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.apiKey) {
      h["Authorization"] = `Bearer ${this.apiKey}`;
    }
    return h;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: Record<string, unknown>
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    try {
      const resp = await fetch(url, {
        method,
        headers: this.headers(),
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (resp.status === 204) {
        return {} as T;
      }

      const data = await resp.json();

      if (!resp.ok) {
        throw new APIError(
          resp.status,
          data.error || "unknown",
          data.message || "request failed"
        );
      }

      return data as T;
    } finally {
      clearTimeout(timer);
    }
  }

  /** Check runtime health. */
  async health(): Promise<HealthResponse> {
    return this.request<HealthResponse>("GET", "/healthz");
  }

  /** List all deployed agents. */
  async listAgents(): Promise<AgentInfo[]> {
    const resp = await this.request<{ agents: AgentInfo[] }>(
      "GET",
      "/v1/agents"
    );
    return resp.agents || [];
  }

  /** Invoke an agent and wait for the complete response. */
  async invoke(
    agentName: string,
    message: string,
    options?: { variables?: Record<string, string>; sessionId?: string }
  ): Promise<InvokeResponse> {
    const body: Record<string, unknown> = { message };
    if (options?.variables) body.variables = options.variables;
    if (options?.sessionId) body.session_id = options.sessionId;

    return this.request<InvokeResponse>(
      "POST",
      `/v1/agents/${agentName}/invoke`,
      body
    );
  }

  /** Create a new conversation session. */
  async createSession(
    agentName: string,
    metadata?: Record<string, string>
  ): Promise<SessionInfo> {
    const body: Record<string, unknown> = {};
    if (metadata) body.metadata = metadata;

    return this.request<SessionInfo>(
      "POST",
      `/v1/agents/${agentName}/sessions`,
      body
    );
  }

  /** Send a message within an existing session. */
  async sendMessage(
    agentName: string,
    sessionId: string,
    message: string
  ): Promise<InvokeResponse> {
    return this.request<InvokeResponse>(
      "POST",
      `/v1/agents/${agentName}/sessions/${sessionId}`,
      { message }
    );
  }

  /** Delete a session and release memory. */
  async deleteSession(agentName: string, sessionId: string): Promise<void> {
    await this.request(
      "DELETE",
      `/v1/agents/${agentName}/sessions/${sessionId}`
    );
  }

  /** Execute a multi-agent pipeline. */
  async runPipeline(
    pipelineName: string,
    trigger: Record<string, unknown>
  ): Promise<PipelineResult> {
    return this.request<PipelineResult>(
      "POST",
      `/v1/pipelines/${pipelineName}/run`,
      { trigger }
    );
  }
}
