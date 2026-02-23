"""AgentSpec SDK streaming support for Python.

Provides an async streaming client that yields SSE events from the
AgentSpec runtime streaming endpoint.
"""

from __future__ import annotations

import asyncio
import json
from dataclasses import dataclass, field
from typing import Any, AsyncGenerator

from .client import (
    AgentSpecError,
    APIError,
    ConnectionError,
    StreamEvent,
    TokenUsage,
)


@dataclass
class StreamResult:
    """Accumulated result from a completed stream."""
    text: str = ""
    tokens: TokenUsage = field(default_factory=TokenUsage)
    turns: int = 0
    duration_ms: int = 0


class AsyncStreamingClient:
    """Async streaming client for the AgentSpec runtime.

    Usage::

        client = AsyncStreamingClient("http://localhost:8080")
        async for event in client.stream("support-bot", "Hello!"):
            if event.event == "text":
                print(event.data["text"], end="", flush=True)
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

    async def stream(
        self,
        agent_name: str,
        message: str,
        variables: dict[str, str] | None = None,
        session_id: str = "",
    ) -> AsyncGenerator[StreamEvent, None]:
        """Stream an agent invocation as async SSE events.

        Yields StreamEvent objects for each server-sent event.
        The final event has event="done" with token usage and timing.
        """
        body: dict[str, Any] = {"message": message}
        if variables:
            body["variables"] = variables
        if session_id:
            body["session_id"] = session_id

        url = f"{self._base_url}/v1/agents/{agent_name}/stream"

        try:
            reader, writer = await asyncio.wait_for(
                asyncio.open_connection(
                    *_parse_host_port(url),
                ),
                timeout=self._timeout,
            )
        except (OSError, asyncio.TimeoutError) as e:
            raise ConnectionError(f"Failed to connect to {url}: {e}")

        try:
            path = _extract_path(url)
            request_body = json.dumps(body).encode()
            headers = f"POST {path} HTTP/1.1\r\n"
            headers += f"Host: {_parse_host(url)}\r\n"
            headers += "Content-Type: application/json\r\n"
            headers += f"Content-Length: {len(request_body)}\r\n"
            if self._api_key:
                headers += f"Authorization: Bearer {self._api_key}\r\n"
            headers += "Accept: text/event-stream\r\n"
            headers += "\r\n"

            writer.write(headers.encode() + request_body)
            await writer.drain()

            # Read HTTP status line
            status_line = await asyncio.wait_for(
                reader.readline(), timeout=self._timeout
            )
            status_code = int(status_line.decode().split(" ")[1])

            # Read headers until blank line
            while True:
                header_line = await reader.readline()
                if header_line == b"\r\n" or header_line == b"\n":
                    break

            if status_code != 200:
                body_data = await reader.read(4096)
                try:
                    err = json.loads(body_data.decode())
                    raise APIError(
                        status_code,
                        err.get("error", "unknown"),
                        err.get("message", "request failed"),
                    )
                except (json.JSONDecodeError, AgentSpecError):
                    raise
                except Exception:
                    raise APIError(status_code, "unknown", "request failed")

            # Parse SSE stream
            event_type = ""
            async for raw_line in reader:
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
                    if event_type == "done":
                        break
                    event_type = ""

        finally:
            writer.close()

    async def stream_text(
        self,
        agent_name: str,
        message: str,
        variables: dict[str, str] | None = None,
        session_id: str = "",
    ) -> StreamResult:
        """Stream an invocation and return the accumulated text result.

        Collects all text chunks and returns a StreamResult with the
        complete text and final token usage.
        """
        result = StreamResult()
        async for event in self.stream(
            agent_name, message, variables=variables, session_id=session_id
        ):
            if event.event == "text":
                result.text += event.data.get("text", "")
            elif event.event == "done":
                tokens_data = event.data.get("tokens", {})
                result.tokens = TokenUsage(
                    input=tokens_data.get("input", 0),
                    output=tokens_data.get("output", 0),
                    total=tokens_data.get("total", 0),
                )
                result.turns = event.data.get("turns", 0)
                result.duration_ms = event.data.get("duration_ms", 0)
        return result


def _parse_host_port(url: str) -> tuple[str, int]:
    """Extract host and port from URL."""
    url = url.replace("http://", "").replace("https://", "")
    host_port = url.split("/")[0]
    if ":" in host_port:
        host, port = host_port.rsplit(":", 1)
        return host, int(port)
    return host_port, 80


def _parse_host(url: str) -> str:
    """Extract host header value from URL."""
    url = url.replace("http://", "").replace("https://", "")
    return url.split("/")[0]


def _extract_path(url: str) -> str:
    """Extract path from URL."""
    url = url.replace("http://", "").replace("https://", "")
    idx = url.find("/")
    if idx == -1:
        return "/"
    return url[idx:]
