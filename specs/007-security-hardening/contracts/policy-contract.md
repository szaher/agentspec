# Contract: Policy Engine Enforcement

**Feature**: 007-security-hardening | **Date**: 2026-03-01

## Overview

Defines the policy engine's `checkRequirement()` behavior for the 4 supported requirement types, evaluation modes, and error reporting.

## Interface: `checkRequirement`

**Package**: `internal/policy`
**File**: `policy.go`

```go
// checkRequirement evaluates a single policy requirement against a resource.
// Returns a list of violations (empty list = pass).
func checkRequirement(rule PolicyRule, resource *ir.Resource, ctx *EvalContext) []Violation

// Violation represents a single policy violation.
type Violation struct {
    Rule        PolicyRule
    Resource    string   // Resource name
    Message     string   // Human-readable description
    Details     []string // Specific items that violated (e.g., unpinned import names)
}
```

**Contract**:
- MUST dispatch on `rule.Action` to one of 4 requirement handlers
- MUST return `[]Violation` with `Rule`, `Resource`, `Message`, and `Details` populated
- MUST return an error (not empty violations) for unknown requirement types
- MUST NOT return `true` unconditionally (current stub behavior must be removed)

## Requirement Type: `pinned imports`

```go
// Validates that ALL import declarations have a version or SHA pin.
```

**Contract**:
- MUST iterate all `resource.References` for import references
- MUST check each import has a version field (semver or SHA)
- MUST list ALL unpinned imports in `Violation.Details` (not just the first)
- Import with `@v1.2.3` or `@sha256:...` is considered pinned

## Requirement Type: `secret`

```go
// Validates that the named secret (from rule.Subject) is configured.
```

**Contract**:
- MUST verify `rule.Subject` (secret name) exists in configured secret resolvers
- MUST check the secret is referenced in the resource's `Attributes`
- MUST report the missing secret name in `Violation.Message`

## Requirement Type: `deny command`

```go
// Validates that no command tool uses the denied binary name.
```

**Contract**:
- MUST compare `rule.Subject` against all command tool configurations in the resource
- MUST match on binary basename (not full path)
- MUST report which tool definition references the denied command

## Requirement Type: `signed packages`

```go
// Validates that imported packages have valid signatures.
```

**Contract**:
- MUST check package manifest for signature field
- If package signing (feature 012) is not yet available: log `"signed packages verification not yet implemented"` at WARNING level and return no violations (graceful degradation)
- When available: MUST verify signature against known public keys

## Evaluation Modes

**File**: `internal/policy/policy.go`

```go
// EvalMode determines how violations are handled.
type EvalMode string

const (
    EvalEnforce EvalMode = "enforce" // Default: violations block apply
    EvalWarn    EvalMode = "warn"    // Violations logged as warnings, apply proceeds
)
```

**Contract**:
- Default mode is `enforce`
- `--policy=warn` flag switches to warn mode
- In `enforce` mode: apply MUST fail if any violations exist
- In `warn` mode: apply MUST proceed; violations logged at WARNING level
- ALL violations are collected and reported together (not just the first)

## Error Output Format

```
Policy violations found (mode: enforce):

  Resource "my-agent":
    - [pinned imports] Import "github.com/example/pkg" is not pinned to a version
    - [pinned imports] Import "github.com/other/lib" is not pinned to a version
    - [deny command] Command tool "cleanup" uses denied binary "rm"

  Resource "helper-agent":
    - [secret] Required secret "api-key" is not configured

Apply blocked: 4 policy violations across 2 resources.
```

**Contract**:
- MUST group violations by resource
- MUST prefix each violation with the requirement type in brackets
- MUST show total violation count and resource count in summary
- In warn mode: replace "Apply blocked" with "Apply proceeding with warnings"
