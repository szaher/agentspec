# Changelog

## [0.2.0] - 2026-02-23

### Changed
- Renamed DSL from "Agentz DSL" to "IntentLang"
- Renamed file extension from `.az` to `.ias`
- Renamed CLI command from `agentz` to `agentspec`
- Renamed definition files to "AgentSpec" and bundles to "AgentPack"
- Go module path (`agentz`) remains unchanged

## [0.1.0] - 2026-02-22

### Added
- Initial release of the AgentSpec toolchain
- Custom `.ias` IntentLang DSL with English-friendly syntax
- Hand-written recursive descent parser with source position error reporting
- Canonical formatter (`agentspec fmt`) with idempotent output
- Two-phase validation (structural + semantic) with "did you mean?" suggestions
- Intermediate Representation (IR) with deterministic JSON serialization
- Content-addressed hashing (SHA-256) for change detection
- Desired-state diff engine with plan/apply lifecycle
- Idempotent apply with partial failure handling
- Drift detection (`agentspec diff`)
- Two adapters: Local MCP and Docker Compose
- Export command for generating platform-specific artifacts
- Multi-environment configuration with overlay merging
- WASM plugin system (wazero) for custom resource types and hooks
- SDK generation for Python, TypeScript, and Go
- Policy engine for security constraints
- Structured event emission with correlation IDs
- Golden fixture integration tests for determinism validation
