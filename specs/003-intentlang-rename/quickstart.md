# Quickstart: Verifying the IntentLang Rename

**Feature**: 003-intentlang-rename
**Date**: 2026-02-23

## Prerequisites

- Go 1.25+ installed
- Features 001-agent-packaging-dsl and 002-ci-pipeline merged to main

## Verification Checklist

### 1. Build the CLI

```bash
go build -o agentspec ./cmd/agentspec
```

Expected: Binary `agentspec` is created successfully.

### 2. Version Output

```bash
./agentspec version
```

Expected: Output shows `agentspec` as the program name (not `agentz`).

### 3. Help Text

```bash
./agentspec --help
```

Expected: All help text references IntentLang, AgentSpec, and `.ias`.

### 4. Validate a .ias File

```bash
./agentspec validate examples/basic-agent/basic-agent.ias
```

Expected: Validation passes with no warnings.

### 5. Validate a .az File (Backward Compatibility)

```bash
# Create a temporary .az file
cp examples/basic-agent/basic-agent.ias /tmp/test.az
./agentspec validate /tmp/test.az
```

Expected: Validation passes but a deprecation warning appears on stderr:
```
Warning: '.az' extension is deprecated. Use '.ias' instead. Run 'agentspec migrate' to rename files.
```

### 6. Format Check All Examples

```bash
for f in examples/*/*.ias; do
  echo "Checking $f..."
  ./agentspec fmt --check "$f"
done
```

Expected: All examples pass format check. No `.az` files exist in `examples/`.

### 7. Migrate Command

```bash
mkdir /tmp/migrate-test
cp examples/basic-agent/basic-agent.ias /tmp/migrate-test/test.az
cd /tmp/migrate-test
agentspec migrate
ls
```

Expected: `test.az` is renamed to `test.ias`. Summary shows 1 file renamed.

### 8. Conflict Detection

```bash
mkdir /tmp/conflict-test
echo 'package "test" version "0.1.0" lang "1.0"' > /tmp/conflict-test/agent.az
echo 'package "test" version "0.1.0" lang "1.0"' > /tmp/conflict-test/agent.ias
./agentspec validate /tmp/conflict-test/agent.az
```

Expected: Error message about conflicting `.az` and `.ias` files.

### 9. State File Migration

```bash
cd /tmp
echo '{}' > .agentz.state.json
./agentspec plan examples/basic-agent/basic-agent.ias
ls .agentspec.state.json
```

Expected: `.agentz.state.json` is renamed to `.agentspec.state.json`. Migration notice on stderr.

### 10. Plugin Directory Fallback

```bash
mkdir -p ~/.agentz/plugins/
touch ~/.agentz/plugins/test-plugin
./agentspec validate examples/basic-agent/basic-agent.ias
```

Expected: Deprecation warning on stderr about `~/.agentz/plugins/` — suggests moving to `~/.agentspec/plugins/`.

### 11. No Stale References

```bash
# Check no .az files remain in examples
find examples -name '*.az' | wc -l
# Expected: 0

# Check Go source for stale references (excluding imports and deprecation code)
grep -r '\.az"' --include='*.go' internal/ cmd/ | grep -v '_test.go' | grep -v 'deprecat'
# Expected: No output (or only intentional backward-compat references)
```

### 12. CI Pipeline

Push the branch and verify the CI workflow passes with:
- Updated glob patterns (`examples/*/*.ias`)
- Binary built as `agentspec`
- All validate/format-check/smoke-test steps green
