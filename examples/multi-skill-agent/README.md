# Multi-Skill Agent

An agent equipped with multiple skills, each with defined input/output schemas and execution commands.

## What This Demonstrates

- **Skill definitions** with typed input/output schemas
- **Multiple `uses skill`** references on a single agent
- **Execution commands** that map skills to external tool binaries

## Definition Structure

### Skills

Each skill declares what it does, what it accepts, and what it returns:

```
skill "web-search" {
  description "Search the web"
  input { query string required }
  output { results string }
  execution command "search-tool"
}
```

- `input` / `output` blocks define the schema with field name, type, and optionality
- `execution command` specifies the binary or script that implements the skill
- `description` is a human-readable summary used in documentation and SDK generation

### Agent with Multiple Skills

```
agent "researcher" {
  uses prompt "research"
  uses skill "web-search"
  uses skill "summarize"
  uses skill "translate"
  model "claude-sonnet-4-20250514"
}
```

An agent can reference any number of skills with `uses skill`. The validator ensures every referenced skill name exists in the file and will suggest corrections for typos.

## How to Run

```bash
# Validate
./agentz validate examples/multi-skill-agent.az

# Plan (shows 5 resources: 1 Prompt + 3 Skills + 1 Agent)
./agentz plan examples/multi-skill-agent.az

# Apply
./agentz apply examples/multi-skill-agent.az --auto-approve

# Export artifacts
./agentz export examples/multi-skill-agent.az --out-dir ./output
```

## Resources Created

| Kind | Name | Description |
|------|------|-------------|
| Prompt | research | Research assistant instructions |
| Skill | web-search | Web search capability |
| Skill | summarize | Text summarization |
| Skill | translate | Text translation |
| Agent | researcher | Agent using all three skills |

## Next Steps

- Expose skills via MCP transport: see [mcp-server-client](../mcp-server-client/)
- Add policies to restrict skill usage: see [data-pipeline](../data-pipeline/)
