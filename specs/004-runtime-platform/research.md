# Research: AgentSpec Runtime Platform

**Branch**: `004-runtime-platform` | **Date**: 2026-02-23

## 1. MCP Client Library for Go

**Decision**: Use the official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`)

**Rationale**: Official SDK maintained by the MCP organization with Google backing. Covers the full MCP specification (2025-11-25 and later). Supports all required transports: stdio, SSE, and streamable-HTTP. Version v1.3.1 is post-v1.0 (stable API guarantee).

**Alternatives considered**:
- `github.com/mark3labs/mcp-go` (v0.44.0): More community adoption (8.2k stars, 1,307 importers) but still pre-v1.0 (API may break). The official SDK is the safer long-term bet for a platform product.

## 2. Anthropic Go SDK

**Decision**: Use the official Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`)

**Rationale**: Official SDK from Anthropic, actively maintained (v1.26.0, 26 releases). Covers all needed features: Messages API, streaming (`NewStreaming`), tool use with JSON schema generation, prompt caching (`BetaToolRunner` for automatic tool use loops). No viable alternatives exist (unofficial SDKs archived).

**Alternatives considered**: None viable.

## 3. Docker SDK for Go

**Decision**: Use `github.com/moby/moby/client` (moby/moby client)

**Rationale**: Stable successor to `docker/docker/client` (now deprecated). Full Docker Engine API access: image build/pull, container lifecycle, health checks, log streaming. Compatible with Docker Engine v29+ (API v1.44+).

**Alternatives considered**:
- `github.com/docker/go-sdk/client` (v0.1.0-alpha012): Higher-level convenience API but still in alpha. Not production-ready.
- `github.com/docker/docker/client`: Deprecated in favor of moby/moby. Not recommended for new projects.

## 4. Kubernetes client-go

**Decision**: Use `k8s.io/client-go` (v0.35.1) with `k8s.io/api` and `k8s.io/apimachinery`

**Rationale**: Standard Kubernetes Go client. Server-Side Apply via `applyconfigurations` package is the modern, declarative approach for manifest application. Dynamic client handles arbitrary manifests. Supports +/- 1 minor version skew with target clusters.

**Alternatives considered**:
- `github.com/manifestival/manifestival`: Convenience layer for "bags of YAML". Could supplement client-go for loading user-provided YAML but not a replacement.

## 5. SSE Streaming

**Decision**: Use Go standard library (`net/http`) for SSE serving

**Rationale**: The SSE protocol is simple enough that ~20 lines of Go handles it. The MCP SDK already includes SSE transport, so no separate SSE dependency is needed for MCP. For the runtime HTTP API streaming endpoint, standard library is sufficient (set `Content-Type: text/event-stream`, flush per event, detect client disconnect).

**Alternatives considered**:
- `github.com/r3labs/sse/v2`: Battle-tested but unmaintained (last release Jan 2023).
- `github.com/tmaxmax/go-sse`: Modern, spec-compliant, pre-v1. Worth revisiting if SSE requirements grow.

## 6. Container Base Image

**Decision**: Use `gcr.io/distroless/static-debian12` for production runtime containers

**Rationale**: Minimal attack surface (no shell, no package manager). Go binaries are statically compiled, so no libc needed. Small image size (~2MB). Industry standard for Go production containers.

**Alternatives considered**:
- `alpine:3.19`: Slightly larger (~7MB) but includes shell for debugging. Good for development containers.
- `scratch`: Even smaller but no CA certificates. Distroless includes CA certs.

## 7. Inline Code Execution

**Decision**: Subprocess execution via `os/exec` with resource limits

**Rationale**: `tool inline` runs user-provided Python/JS code. Subprocess isolation via `os/exec.CommandContext` with context timeout. Memory limits via `syscall.Setrlimit` on Linux (advisory on macOS). Filesystem access restricted via working directory isolation. Environment variables and secrets passed explicitly.

**Alternatives considered**:
- WASM sandbox (wazero): Only supports WASM modules, not raw Python/JS. Would require compiling user code to WASM, which is impractical.
- Container sandbox: Too heavy for per-tool-call isolation. Acceptable for deployment-level isolation but not inline code.

## Dependency Summary

| Component | Library | Version | Import Path |
| --------- | ------- | ------- | ----------- |
| MCP Client | Official MCP SDK | v1.3.1 | `github.com/modelcontextprotocol/go-sdk/mcp` |
| LLM Client | Anthropic SDK | v1.26.0 | `github.com/anthropics/anthropic-sdk-go` |
| Docker | moby/moby client | v29.x | `github.com/moby/moby/client` |
| Kubernetes | client-go | v0.35.1 | `k8s.io/client-go` |
| SSE | Go stdlib | Go 1.25+ | `net/http` |
| WASM | wazero | v1.11.0 | `github.com/tetratelabs/wazero` (existing) |
| CLI | cobra | v1.10.2 | `github.com/spf13/cobra` (existing) |
