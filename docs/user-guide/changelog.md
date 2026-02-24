# Changelog

All notable changes to AgentSpec are documented on this page.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

### Added

- Comprehensive documentation site with Material for MkDocs
- IntentLang 2.0 language reference for all 13 resource types
- Getting started guide with quickstart tutorial
- 7 agentic architecture use-case guides (ReAct, Plan-and-Execute, Reflexion, Router, Map-Reduce, Pipeline, Delegation)
- Deployment guides for Process, Docker, Docker Compose, and Kubernetes targets
- CLI command reference for all 15 subcommands
- HTTP API reference documentation
- SDK documentation for Python, TypeScript, and Go
- Developer guide with architecture, contributing, and extending documentation
- IntentLang syntax highlighting via custom Pygments lexer
- Automated `.ias` code block validation in documentation via Go integration tests

---

## [0.3.0] -- IntentLang Rename

### Changed

- Renamed language from "AgentLang" to "IntentLang"
- Changed file extension from `.az` to `.ias`
- Renamed CLI binary from `agentz` to `agentspec`
- Renamed state file from `.agentz.state.json` to `.agentspec.state.json`
- Updated plugin directory from `~/.agentz/plugins/` to `~/.agentspec/plugins/`

### Added

- Automatic state file migration from `.agentz.state.json`
- `agentspec migrate` command for file conversion

---

## [0.2.0] -- CI Pipeline

### Added

- GitHub Actions CI workflow
- Automated linting with `golangci-lint`
- Test pipeline with Go test runner
- Build validation on pull requests
- Example file validation in CI

---

## [0.1.0] -- Initial Release

### Added

- IntentLang 2.0 parser with full syntax support
- 13 resource types: `agent`, `prompt`, `skill`, `tool`, `deploy`, `pipeline`, `type`, `server`, `client`, `secret`, `environment`, `policy`, `plugin`
- 4 tool variants: `mcp`, `http`, `command`, `inline`
- 5 deployment targets: `process`, `docker`, `docker-compose`, `kubernetes`
- 5 agent strategies: `react`, `plan-and-execute`, `reflexion`, `router`, `map-reduce`
- `agentspec validate` command for syntax and semantic validation
- `agentspec fmt` command for code formatting
- `agentspec plan` command for deployment planning
- `agentspec apply` command for deployment execution
- Desired-state deployment model with state tracking
- Package header with semantic versioning
- WASM plugin system via wazero

---

## See Also

- [Migration Guide](migration.md) -- Migrating from IntentLang 1.0 to 2.0
- [Getting Started](getting-started/index.md) -- Installation and setup
