/**
 * AgentSpec SDK streaming support for TypeScript.
 *
 * Provides an SSE-based streaming client that yields events
 * from the AgentSpec runtime streaming endpoint.
 */

import type { StreamEvent, TokenUsage, ClientOptions } from "./client";
import { AgentSpecError, APIError } from "./client";

/** Accumulated result from a completed stream. */
export interface StreamResult {
  text: string;
  tokens: TokenUsage;
  turns: number;
  duration_ms: number;
}

/** Callback for streaming events. */
export type StreamCallback = (event: StreamEvent) => void;

/**
 * Stream an agent invocation using fetch + ReadableStream (SSE).
 *
 * @example
 * ```ts
 * for await (const event of streamInvocation(
 *   { baseUrl: "http://localhost:8080" },
 *   "support-bot",
 *   "Hello!"
 * )) {
 *   if (event.event === "text") {
 *     process.stdout.write(event.data.text as string);
 *   }
 * }
 * ```
 */
export async function* streamInvocation(
  options: ClientOptions,
  agentName: string,
  message: string,
  invokeOptions?: {
    variables?: Record<string, string>;
    sessionId?: string;
  }
): AsyncGenerator<StreamEvent, void, undefined> {
  const baseUrl = (options.baseUrl || "http://localhost:8080").replace(
    /\/$/,
    ""
  );
  const url = `${baseUrl}/v1/agents/${agentName}/stream`;

  const body: Record<string, unknown> = { message };
  if (invokeOptions?.variables) body.variables = invokeOptions.variables;
  if (invokeOptions?.sessionId) body.session_id = invokeOptions.sessionId;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "text/event-stream",
  };
  if (options.apiKey) {
    headers["Authorization"] = `Bearer ${options.apiKey}`;
  }

  const controller = new AbortController();
  const timeout = options.timeout || 120000;
  const timer = setTimeout(() => controller.abort(), timeout);

  let resp: Response;
  try {
    resp = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
      signal: controller.signal,
    });
  } catch (err) {
    clearTimeout(timer);
    throw new AgentSpecError(
      `Failed to connect to ${url}: ${(err as Error).message}`
    );
  }

  if (!resp.ok) {
    clearTimeout(timer);
    let errData: Record<string, string>;
    try {
      errData = await resp.json();
    } catch {
      errData = { error: "unknown", message: "request failed" };
    }
    throw new APIError(
      resp.status,
      errData.error || "unknown",
      errData.message || "request failed"
    );
  }

  if (!resp.body) {
    clearTimeout(timer);
    throw new AgentSpecError("Response body is null, streaming not supported");
  }

  try {
    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let eventType = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("event: ")) {
          eventType = line.slice(7).trim();
        } else if (line.startsWith("data: ")) {
          const dataStr = line.slice(6);
          let data: Record<string, unknown>;
          try {
            data = JSON.parse(dataStr);
          } catch {
            data = { raw: dataStr };
          }
          yield { event: eventType, data };
          if (eventType === "done") {
            return;
          }
          eventType = "";
        }
      }
    }
  } finally {
    clearTimeout(timer);
  }
}

/**
 * Stream an agent invocation and collect the full text result.
 *
 * @example
 * ```ts
 * const result = await streamText(
 *   { baseUrl: "http://localhost:8080" },
 *   "support-bot",
 *   "Hello!"
 * );
 * console.log(result.text);
 * ```
 */
export async function streamText(
  options: ClientOptions,
  agentName: string,
  message: string,
  invokeOptions?: {
    variables?: Record<string, string>;
    sessionId?: string;
  }
): Promise<StreamResult> {
  const result: StreamResult = {
    text: "",
    tokens: { input: 0, output: 0, cache_read: 0, total: 0 },
    turns: 0,
    duration_ms: 0,
  };

  for await (const event of streamInvocation(
    options,
    agentName,
    message,
    invokeOptions
  )) {
    if (event.event === "text") {
      result.text += (event.data.text as string) || "";
    } else if (event.event === "done") {
      const tokens = event.data.tokens as Record<string, number> | undefined;
      if (tokens) {
        result.tokens = {
          input: tokens.input || 0,
          output: tokens.output || 0,
          cache_read: tokens.cache_read || 0,
          total: tokens.total || 0,
        };
      }
      result.turns = (event.data.turns as number) || 0;
      result.duration_ms = (event.data.duration_ms as number) || 0;
    }
  }

  return result;
}
