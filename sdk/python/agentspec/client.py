"""AgentSpec SDK client for Python.

Provides typed methods for invoking agents, streaming responses,
and managing sessions via the AgentSpec runtime HTTP API.
"""

from __future__ import annotations

import json
import urllib.request
import urllib.error
from dataclasses import dataclass, field
from typing import Any, Generator, Optional


@dataclass
class TokenUsage:
    """Token usage statistics from an invocation."""
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
    """A single event from a streaming invocation."""
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
    """Base error for AgentSpec SDK."""
    pass


class APIError(AgentSpecError):
    """Error returned by the AgentSpec runtime API."""

    def __init__(self, status_code: int, error_code: str, message: str):
        super().__init__(f"API error {status_code} ({error_code}): {message}")
        self.status_code = status_code
        self.error_code = error_code


class ConnectionError(AgentSpecError):
    """Error connecting to the AgentSpec runtime."""
    pass


class AgentSpecClient:
    """Client for the AgentSpec runtime HTTP API.

    Usage::

        client = AgentSpecClient("http://localhost:8080")
        response = client.invoke("support-bot", "Hello!")
        print(response.output)
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        api_key: str = "",
        timeout: int = 120,
    ):
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        headers = {"Content-Type": "application/json"}
        if self._api_key:
            headers["Authorization"] = f"Bearer {self._api_key}"
        return headers

    def _request(
        self, method: str, path: str, body: dict[str, Any] | None = None
    ) -> dict[str, Any]:
        url = f"{self._base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(
            url, data=data, headers=self._headers(), method=method
        )
        try:
            with urllib.request.urlopen(req, timeout=self._timeout) as resp:
                if resp.status == 204:
                    return {}
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            try:
                err_body = json.loads(e.read().decode())
                raise APIError(
                    e.code,
                    err_body.get("error", "unknown"),
                    err_body.get("message", str(e)),
                )
            except (json.JSONDecodeError, AgentSpecError):
                raise
            except Exception:
                raise APIError(e.code, "unknown", str(e))
        except urllib.error.URLError as e:
            raise ConnectionError(f"Failed to connect to {url}: {e}")

    def health(self) -> dict[str, Any]:
        """Check runtime health."""
        return self._request("GET", "/healthz")

    def list_agents(self) -> list[AgentInfo]:
        """List all deployed agents."""
        resp = self._request("GET", "/v1/agents")
        agents = []
        for a in resp.get("agents", []):
            agents.append(AgentInfo(
                name=a.get("name", ""),
                fqn=a.get("fqn", ""),
                model=a.get("model", ""),
                strategy=a.get("strategy", ""),
                status=a.get("status", ""),
                skills=a.get("skills", []),
                active_sessions=a.get("active_sessions", 0),
            ))
        return agents

    def invoke(
        self,
        agent_name: str,
        message: str,
        variables: dict[str, str] | None = None,
        session_id: str = "",
    ) -> InvokeResponse:
        """Invoke an agent and wait for the complete response."""
        body: dict[str, Any] = {"message": message}
        if variables:
            body["variables"] = variables
        if session_id:
            body["session_id"] = session_id

        resp = self._request("POST", f"/v1/agents/{agent_name}/invoke", body)

        tool_calls = []
        for tc in resp.get("tool_calls", []):
            tool_calls.append(ToolCall(
                id=tc.get("id", ""),
                tool_name=tc.get("tool_name", ""),
                input=tc.get("input", {}),
                output=tc.get("output"),
                duration_ms=tc.get("duration_ms", 0),
                error=tc.get("error", ""),
            ))

        tokens_data = resp.get("tokens", {})
        tokens = TokenUsage(
            input=tokens_data.get("input", 0),
            output=tokens_data.get("output", 0),
            cache_read=tokens_data.get("cache_read", 0),
            total=tokens_data.get("total", 0),
        )

        return InvokeResponse(
            output=resp.get("output", ""),
            tool_calls=tool_calls,
            tokens=tokens,
            turns=resp.get("turns", 0),
            duration_ms=resp.get("duration_ms", 0),
            session_id=resp.get("session_id", ""),
        )

    def stream(
        self,
        agent_name: str,
        message: str,
        variables: dict[str, str] | None = None,
        session_id: str = "",
    ) -> Generator[StreamEvent, None, None]:
        """Invoke an agent with streaming response.

        Yields StreamEvent objects for each SSE event.
        """
        body: dict[str, Any] = {"message": message}
        if variables:
            body["variables"] = variables
        if session_id:
            body["session_id"] = session_id

        url = f"{self._base_url}/v1/agents/{agent_name}/stream"
        data = json.dumps(body).encode()
        req = urllib.request.Request(
            url, data=data, headers=self._headers(), method="POST"
        )

        try:
            resp = urllib.request.urlopen(req, timeout=self._timeout)
        except urllib.error.HTTPError as e:
            try:
                err_body = json.loads(e.read().decode())
                raise APIError(
                    e.code,
                    err_body.get("error", "unknown"),
                    err_body.get("message", str(e)),
                )
            except (json.JSONDecodeError, AgentSpecError):
                raise
            except Exception:
                raise APIError(e.code, "unknown", str(e))
        except urllib.error.URLError as e:
            raise ConnectionError(f"Failed to connect to {url}: {e}")

        yield from _parse_sse(resp)

    def create_session(
        self,
        agent_name: str,
        metadata: dict[str, str] | None = None,
    ) -> SessionInfo:
        """Create a new conversation session."""
        body: dict[str, Any] = {}
        if metadata:
            body["metadata"] = metadata
        resp = self._request("POST", f"/v1/agents/{agent_name}/sessions", body)
        return SessionInfo(
            session_id=resp.get("session_id", ""),
            agent=resp.get("agent", ""),
            created_at=resp.get("created_at", ""),
        )

    def send_message(
        self,
        agent_name: str,
        session_id: str,
        message: str,
    ) -> InvokeResponse:
        """Send a message within an existing session."""
        body = {"message": message}
        resp = self._request(
            "POST", f"/v1/agents/{agent_name}/sessions/{session_id}", body
        )
        return InvokeResponse(
            output=resp.get("output", ""),
            turns=resp.get("turns", 0),
            duration_ms=resp.get("duration_ms", 0),
            session_id=resp.get("session_id", ""),
        )

    def delete_session(self, agent_name: str, session_id: str) -> None:
        """Delete a session and release memory."""
        self._request(
            "DELETE", f"/v1/agents/{agent_name}/sessions/{session_id}"
        )

    def run_pipeline(
        self,
        pipeline_name: str,
        trigger: dict[str, Any],
    ) -> PipelineResult:
        """Execute a multi-agent pipeline."""
        resp = self._request(
            "POST", f"/v1/pipelines/{pipeline_name}/run", {"trigger": trigger}
        )

        steps = {}
        for name, step_data in resp.get("steps", {}).items():
            steps[name] = PipelineStepResult(
                agent=step_data.get("agent", ""),
                output=step_data.get("output"),
                duration_ms=step_data.get("duration_ms", 0),
                status=step_data.get("status", ""),
                error=step_data.get("error", ""),
            )

        tokens_data = resp.get("tokens", {})

        return PipelineResult(
            pipeline=resp.get("pipeline", ""),
            status=resp.get("status", ""),
            steps=steps,
            total_duration_ms=resp.get("total_duration_ms", 0),
            tokens=TokenUsage(total=tokens_data.get("total", 0)),
        )


def _parse_sse(resp) -> Generator[StreamEvent, None, None]:
    """Parse Server-Sent Events from an HTTP response."""
    event_type = ""
    for raw_line in resp:
        line = raw_line.decode("utf-8").rstrip("\n\r")
        if line.startswith("event: "):
            event_type = line[7:]
        elif line.startswith("data: "):
            data_str = line[6:]
            try:
                data = json.loads(data_str)
            except json.JSONDecodeError:
                data = {"raw": data_str}
            yield StreamEvent(event=event_type, data=data)
            event_type = ""
        elif line == "":
            continue
