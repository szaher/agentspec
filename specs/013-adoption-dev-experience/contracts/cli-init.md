# CLI Contract: `agentspec init`

## Synopsis

```
agentspec init [--template <name>] [--list-templates] [--output-dir <path>] [--name <name>]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--template` | string | (none) | Template name. If omitted, interactive selector is shown. |
| `--list-templates` | bool | false | List available templates with descriptions and exit. |
| `--output-dir` | string | `.` | Target directory for scaffolded project. |
| `--name` | string | (template name) | Project/package name. |

## Behavior

### No flags (interactive mode)

```
$ agentspec init
Choose a starter template:

  Beginner:
    1. basic-chatbot         Simple conversational agent
    2. support-bot           Customer support with tools

  Intermediate:
    3. rag-assistant         RAG with document retrieval
    4. incident-response     Incident triage and response

  Advanced:
    5. research-swarm        Multi-agent research coordination
    6. multi-agent-router    Request routing across agents

Select template [1-6]: _
```

### `--list-templates`

```
$ agentspec init --list-templates
Available templates:

  basic-chatbot          Simple conversational agent                    [beginner]
  support-bot            Customer support with tools                   [beginner]
  rag-assistant          RAG with document retrieval                   [intermediate]
  incident-response      Incident triage and response                  [intermediate]
  research-swarm         Multi-agent research coordination             [advanced]
  multi-agent-router     Request routing across agents                 [advanced]
```

### `--template <name>`

```
$ agentspec init --template support-bot
Created project in ./support-bot/

Files:
  support-bot/support-bot.ias    Agent definition
  support-bot/README.md          Setup and run instructions

Next steps:
  1. Set required environment variables:
     export ANTHROPIC_API_KEY="your-key-here"
  2. Validate: agentspec validate support-bot/support-bot.ias
  3. Run:      agentspec run support-bot/support-bot.ias
```

### Existing files detected

```
$ agentspec init --template support-bot
Warning: support-bot/support-bot.ias already exists.
Overwrite? [y/N]: _
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unknown template, file conflict (user declined), or I/O error |

## Output Directory Structure

Each template scaffolds a directory containing:

```
<project-name>/
├── <project-name>.ias    # Agent definition with env var references
└── README.md             # Description, prerequisites, run instructions
```
