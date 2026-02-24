# Pull Request Guidelines

This guide covers the process for contributing code changes to the AgentSpec project, from branch creation through merge.

## Branch Naming

Use descriptive branch names with a prefix that indicates the type of change:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `feat/` | New feature | `feat/redis-session-store` |
| `fix/` | Bug fix | `fix/parser-duplicate-detection` |
| `refactor/` | Code refactoring | `refactor/adapter-registry` |
| `docs/` | Documentation | `docs/plugin-guide` |
| `test/` | Test additions or fixes | `test/ir-lowering-coverage` |
| `chore/` | Maintenance, deps, CI | `chore/update-wazero` |

Branch names should be lowercase with hyphens separating words.

## Workflow

### 1. Create a Branch

```bash
git checkout main
git pull origin main
git checkout -b feat/my-feature
```

### 2. Make Changes

- Write code following the [Code Style](code-style.md) guide.
- Add or update tests for all changed behavior.
- Run the full test suite before committing:

```bash
go test ./... -count=1
```

- Run the linter:

```bash
golangci-lint run ./...
```

### 3. Commit

Write clear, descriptive commit messages.

#### Commit Message Format

```text
<type>: <short summary>

<optional body with more detail>
```

Types match branch prefixes: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`.

Examples:

```text
feat: add Redis session store

Implement session.RedisStore backed by Redis with configurable
TTL and key prefix. Falls back to in-memory store when Redis
is unavailable.
```

```text
fix: handle duplicate agent names in parser

The parser was silently ignoring duplicate agent declarations.
Now it reports a ParseError with a hint suggesting unique names.
```

#### Guidelines

- Keep the first line under 72 characters.
- Use the imperative mood: "add feature" not "added feature".
- Explain *why* in the body, not just *what*.
- Reference issue numbers when applicable: `Fixes #42`.

### 4. Push and Create PR

```bash
git push -u origin feat/my-feature
```

Create the pull request on GitHub targeting the `main` branch.

## PR Description

Every PR should include:

### Summary

A brief description of what the PR does and why. Link to any related issues.

### Changes

Bullet list of specific changes:

- Added `session.RedisStore` implementing the `session.Store` interface
- Updated `runtime.New()` to accept a store configuration option
- Added integration tests with a Redis test container

### Test Plan

How was this tested:

- Unit tests for `RedisStore` methods
- Integration test with real Redis via testcontainers
- Manual verification with `agentspec run` against a sample agent

## CI Checks

The CI pipeline runs automatically on every PR. All checks must pass before merge.

| Check | What It Does |
|-------|-------------|
| **Build** | `go build ./cmd/agentspec` -- Verifies the project compiles |
| **Test** | `go test ./... -count=1` -- Runs all unit and integration tests |
| **Lint** | `golangci-lint run ./...` -- Checks code style and correctness |
| **Format** | Verifies `gofmt` has been applied |

If a check fails:

1. Read the CI logs to understand the failure.
2. Fix the issue locally.
3. Push the fix. CI will re-run automatically.

## Code Review

### For Authors

- Keep PRs focused. One logical change per PR. If a PR touches unrelated areas, split it.
- Respond to reviewer comments promptly.
- If you disagree with feedback, explain your reasoning rather than silently ignoring it.
- Mark resolved conversations.

### For Reviewers

Review focus areas:

- **Correctness** -- Does the code do what it claims?
- **Tests** -- Are new behaviors tested? Are edge cases covered?
- **Naming** -- Are names clear and consistent with existing patterns?
- **Error handling** -- Are errors wrapped with context? Are all error paths tested?
- **Interface design** -- Are new interfaces minimal? Do they follow existing patterns?
- **Documentation** -- Are exported symbols documented? Is the package doc updated?

### Approval

PRs require at least one approval from a maintainer before merge.

## Merge Strategy

The project uses **squash and merge** for most PRs. This keeps the main branch history clean with one commit per logical change.

For large, multi-commit PRs where individual commits are meaningful, a regular merge commit may be used instead.

## Changelog

When a PR adds user-facing changes, update the changelog:

- New features or commands
- Breaking changes to the `.ias` syntax or CLI interface
- Bug fixes that affect user-visible behavior
- Dependency version bumps that affect functionality

Internal refactors, test-only changes, and documentation updates typically do not need changelog entries.

## Checklist

Before requesting review, verify:

- [ ] Code compiles: `go build ./cmd/agentspec`
- [ ] All tests pass: `go test ./... -count=1`
- [ ] Linter is clean: `golangci-lint run ./...`
- [ ] New code has tests
- [ ] Exported symbols have doc comments
- [ ] Commit messages are clear and follow the format
- [ ] PR description includes summary, changes, and test plan
