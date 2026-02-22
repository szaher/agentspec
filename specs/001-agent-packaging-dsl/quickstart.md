# Quickstart: Golden Path Demo

**Goal**: From a fresh clone to a working deployment in under
15 minutes.

## Prerequisites

- Go 1.25+ installed
- Docker (for Docker Compose adapter demo, optional)

## Step 1: Build the CLI

```bash
git clone <repo-url>
cd agentz
go build -o agentz ./cmd/agentz
```

Verify:

```bash
./agentz version
# agentz version 0.1.0 (lang 1.0, ir 1.0)
```

## Step 2: Write Your First Definition

Create `demo.az`:

```
package "demo" version "0.1.0" lang "1.0"

prompt "greeting" {
  content "You are a helpful research assistant.
           Answer questions clearly and concisely."
}

skill "web-search" {
  description "Search the web for information"
  input { query string required }
  output { results string }
  execution command "search-tool"
}

skill "summarize" {
  description "Summarize a block of text"
  input { text string required }
  output { summary string }
  execution command "summarize-tool"
}

agent "research-assistant" {
  uses prompt "greeting"
  uses skill "web-search"
  uses skill "summarize"
  model "claude-sonnet-4-20250514"
}

binding "local" adapter "local-mcp" {
  default true
  output_dir "./deploy"
}
```

## Step 3: Format and Validate

```bash
./agentz fmt demo.az
./agentz validate demo.az
# No errors
```

Introduce an error to see diagnostics:

```bash
# Change "web-search" to "web-serch" in the agent block
./agentz validate demo.az
# demo.az:18:14: error: skill "web-serch" not found
#   hint: did you mean "web-search"?
```

Fix and re-validate.

## Step 4: Plan Changes

```bash
./agentz plan
```

Expected output:

```
Plan: 4 resources to create, 0 to update, 0 to delete

  + Agent/research-assistant
  + Prompt/greeting
  + Skill/web-search
  + Skill/summarize

Target: local-mcp (binding "local")
```

Verify determinism:

```bash
./agentz plan --format json > plan1.json
./agentz plan --format json > plan2.json
diff plan1.json plan2.json
# No differences
```

## Step 5: Apply

```bash
./agentz apply --auto-approve
```

Expected output:

```
Applying 4 resources to local-mcp...
  ✓ Prompt/greeting         created
  ✓ Skill/web-search        created
  ✓ Skill/summarize         created
  ✓ Agent/research-assistant created

4 created, 0 updated, 0 failed
State saved to .agentz.state.json
```

## Step 6: Verify Idempotency

```bash
./agentz apply --auto-approve
```

Expected output:

```
No changes. Infrastructure is up-to-date.
```

## Step 7: Export to Docker Compose

Add a second binding to `demo.az`:

```
binding "compose" adapter "docker-compose" {
  output_dir "./compose-deploy"
}
```

```bash
./agentz export --target compose
ls compose-deploy/
# docker-compose.yml  config/
```

## Step 8: Use the SDK (Python)

```bash
./agentz sdk generate --lang python --out-dir ./sdk/python
cd sdk/python
pip install -e .
```

```python
from agentz import AgentzClient

client = AgentzClient(state_file="../.agentz.state.json")
agents = client.list_agents()
for agent in agents:
    print(f"{agent.name} ({agent.status})")

# research-assistant (applied)
```

## Verification Checklist

- [ ] `agentz fmt` produces identical output on re-run
- [ ] `agentz validate` catches errors with file:line:col + hint
- [ ] `agentz plan` produces byte-identical output across runs
- [ ] `agentz apply` creates resources on first run
- [ ] `agentz apply` reports no changes on second run
- [ ] `agentz export --target compose` produces Docker artifacts
- [ ] Python SDK lists the applied agent
