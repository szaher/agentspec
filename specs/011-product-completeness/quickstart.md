# Quickstart: Product Completeness & UX

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## Scenario 1: Compile Agent with Functional Tools

```bash
# Create an agent with HTTP and command tools
cat > tools-demo.ias <<'EOF'
package "tools-demo" version "0.1.0" lang "3.0"

prompt "system" {
  content "You are a helpful assistant with web search and file listing capabilities."
}

skill "web-search" {
  description "Search the web"
  input { query string required }
  output { results string }
  tool http { url "https://api.example.com/search" method "GET" }
}

skill "list-files" {
  description "List files in a directory"
  input { path string required }
  output { files string }
  tool command { binary "ls" args "-la" }
}

agent "assistant" {
  uses prompt "system"
  uses skill "web-search"
  uses skill "list-files"
  model "claude-sonnet-4-20250514"
}
EOF

# Compile to CrewAI — tools should have real HTTP/subprocess calls
agentspec compile tools-demo.ias --target crewai --out-dir ./crewai-output

# Verify no "not implemented" stubs in generated code
grep -r "not implemented" ./crewai-output/ && echo "FAIL: stubs found" || echo "PASS: no stubs"
```

## Scenario 2: Live Eval

```bash
# Create agent with eval cases
cat > eval-demo.ias <<'EOF'
package "eval-demo" version "0.1.0" lang "3.0"

prompt "math" {
  content "You are a math assistant. Answer with just the number."
}

agent "math-agent" {
  uses prompt "math"
  model "claude-haiku-4-5-20251001"

  eval "basic-addition" {
    input "What is 2 + 2?"
    expected contains "4"
    tags ["math", "basic"]
  }
}
EOF

# Run with live LLM invocation
agentspec eval eval-demo.ias --live

# Expected output:
# Agent: math-agent
#   PASS  basic-addition: What is 2 + 2?
# Summary: 1/1 passed (100%)
```

## Scenario 3: Dev Server with File Watching

```bash
# Start the server (NEW: 'run' now starts server, was 'dev')
agentspec run hello.ias --port 8080 --ui

# In another terminal, edit hello.ias
# Change should be detected within 500ms (fsnotify, not polling)

# One-shot invocation (NEW: 'dev' now does one-shot, was 'run')
agentspec dev hello.ias --input "Hello, world!"
```

## Scenario 4: Frontend States

```bash
# Start server
agentspec run hello.ias --port 8080 --ui

# Open browser to http://localhost:8080
# Expected flow:
# 1. Loading spinner appears briefly
# 2. Agents load → welcome card with instructions shown
# 3. Type message → welcome disappears, chat begins
# 4. Stop server → error banner with retry button appears
# 5. Restart server → click retry → reconnects
```

## Scenario 5: Honest Feature Flags

```bash
# --sign should now error, not silently continue
agentspec publish --sign
# Expected: Error: Package signing is not yet available. Publish without --sign.
# Exit code: 1 (not 0)
```
