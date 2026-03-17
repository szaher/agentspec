# Data Model: Product Completeness & UX

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## Overview

This feature modifies existing entities rather than introducing new ones. No new database tables, state file schema changes, or persistent data structures are required.

## Modified Entities

### 1. CLI Command Registry

**Location**: `cmd/agentspec/main.go` (newRootCmd)

Current: 20 commands registered via `root.AddCommand()`.

Changes:
- `run` command: Swap to server behavior (current `dev` logic)
- `dev` command: Swap to one-shot invocation (current `run` logic)
- Add deprecation alias commands that emit warnings and delegate

### 2. Eval Invoker Interface

**Location**: `cmd/agentspec/eval.go`

Current interface (implicit):
```go
type Invoker interface {
    Invoke(ctx context.Context, agentName, input string) (string, error)
}
```

New: Add `liveInvoker` implementing this interface using `loop.ReActStrategy`.

### 3. Compiler Target Tool Output

**Location**: `internal/compiler/targets/*.go`

Each target's `generateTools()` method. No schema change — just richer code generation for existing tool types (HTTP, command, inline).

### 4. Frontend State Machine

**Location**: `internal/frontend/web/app.js`

New UI states:
- `loading`: Shown during initial agent fetch
- `error`: Shown on connection failure, includes retry action
- `empty`: Shown when no messages exist, includes usage instructions
- `ready`: Normal interactive state (existing)

### 5. File Watcher

**Location**: `cmd/agentspec/dev.go` (after rename, this becomes the server command)

Replace: `time.NewTicker` + `filepath.Walk` polling loop
With: `fsnotify.Watcher` event-based watching with `.ias` filter and 100ms debounce

## No New Entities

- No new IR resource types
- No state file schema changes
- No new API endpoints
- No database migrations
