# Testing Guide

This guide covers the testing strategy, conventions, and tools used in the AgentSpec codebase.

## Running Tests

### All Tests

```bash
go test ./... -count=1
```

The `-count=1` flag bypasses the Go test cache, ensuring every test runs fresh.

### Single Package

```bash
go test ./internal/parser/ -count=1 -v
```

The `-v` flag enables verbose output, showing individual test names and results.

### Single Test

```bash
go test ./internal/parser/ -count=1 -v -run TestParseAgent
```

### Integration Tests

Integration tests live in `integration_tests/` and may require external dependencies (Docker, etc.):

```bash
go test ./integration_tests/ -count=1 -v
```

### Race Detection

Run tests with the race detector for concurrent code:

```bash
go test ./... -count=1 -race
```

### Coverage

Generate a coverage report:

```bash
go test ./... -count=1 -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Structure

### File Naming

Test files follow Go conventions:

- `parser_test.go` for tests of `parser.go`
- `adapter_test.go` for tests of `adapter.go`

### Test Naming

Test functions use descriptive names that explain the scenario:

```go
func TestParseAgent_WithModel(t *testing.T)
func TestParseAgent_DuplicateName(t *testing.T)
func TestComputePlan_NewResource(t *testing.T)
func TestComputePlan_HashChanged(t *testing.T)
func TestComputePlan_ResourceDeleted(t *testing.T)
```

Pattern: `Test<Function>_<Scenario>`

### Table-Driven Tests

Most tests use the table-driven pattern for comprehensive coverage:

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:  "valid agent",
            input: `package "test" { ... }`,
        },
        {
            name:    "missing package",
            input:   `agent "x" { }`,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, errs := parser.Parse(tt.input, "test.ias")
            if (len(errs) > 0) != tt.wantErr {
                t.Errorf("Parse() errors = %v, wantErr %v", errs, tt.wantErr)
            }
        })
    }
}
```

## Testdata Directory

Test fixtures are stored in `testdata/` directories within each package:

```text
internal/parser/
  parser.go
  parser_test.go
  testdata/
    valid_agent.ias
    valid_pipeline.ias
    invalid_syntax.ias
    expected_ast.json
```

Load test fixtures using `os.ReadFile`:

```go
func TestParseFile(t *testing.T) {
    input, err := os.ReadFile("testdata/valid_agent.ias")
    if err != nil {
        t.Fatal(err)
    }
    ast, errs := parser.Parse(string(input), "valid_agent.ias")
    // ...
}
```

## Golden Tests

Golden tests compare output against saved expected results. This pattern is used for:

- Parser output (AST JSON)
- Formatter output (canonical `.ias`)
- IR lowering output (IR JSON)
- Plan output (action list)

### Writing a Golden Test

```go
func TestFormat_Golden(t *testing.T) {
    input, _ := os.ReadFile("testdata/input.ias")
    expected, _ := os.ReadFile("testdata/expected.ias")

    got := formatter.Format(string(input))

    if diff := cmp.Diff(string(expected), got); diff != "" {
        t.Errorf("Format() mismatch (-want +got):\n%s", diff)
    }
}
```

### Updating Golden Files

When the expected output changes intentionally, update the golden files:

```bash
go test ./internal/formatter/ -count=1 -v -update
```

If the package supports the `-update` flag, golden files are overwritten with the current output. Otherwise, manually copy the output.

## Test Helpers

### go-cmp

The project uses `github.com/google/go-cmp` (v0.7.0) for structural comparison:

```go
import "github.com/google/go-cmp/cmp"

if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("mismatch (-want +got):\n%s", diff)
}
```

This produces clear, readable diffs for complex structures.

### t.Helper()

Helper functions call `t.Helper()` to ensure error messages point to the calling test, not the helper:

```go
func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### t.TempDir()

Use `t.TempDir()` for tests that need temporary files. The directory is automatically cleaned up:

```go
func TestExport(t *testing.T) {
    dir := t.TempDir()
    err := adapter.Export(ctx, resources, dir)
    // ... verify files in dir
}
```

## Testing Adapters

Adapter tests typically use test doubles or mock implementations:

```go
type mockAdapter struct {
    applyFn func(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error)
}

func (m *mockAdapter) Apply(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error) {
    return m.applyFn(ctx, actions)
}
```

Integration tests that require Docker or Kubernetes should be guarded with build tags or environment checks:

```go
func TestDockerAdapter(t *testing.T) {
    if os.Getenv("AGENTSPEC_TEST_DOCKER") == "" {
        t.Skip("set AGENTSPEC_TEST_DOCKER=1 to run Docker tests")
    }
    // ...
}
```

## Testing Plugins

Plugin tests use pre-compiled WASM test fixtures stored in `testdata/`:

```go
func TestPluginValidation(t *testing.T) {
    ctx := context.Background()
    host, _ := plugins.NewHost(ctx)
    defer host.Close(ctx)

    plugin, _ := host.LoadPlugin(ctx, "testdata/test_validator.wasm")
    // ...
}
```

## CI Integration

The CI pipeline (GitHub Actions) runs:

1. `go test ./... -count=1 -race` -- All tests with race detection
2. `golangci-lint run ./...` -- Linting
3. `go build ./cmd/agentspec` -- Build verification

Tests must pass and the linter must report no issues before a PR can be merged.
