# AgentSpec IntentLang — VS Code Extension

Language support for [IntentLang 3.0](../docs/) (`.ias`) files used by [AgentSpec](../).

## Features

- **Syntax highlighting** — keywords, blocks, attributes, control flow, strings, comments
- **Code snippets** — 20+ templates for agents, prompts, skills, config, eval, control flow, Ollama agents
- **Autocomplete** — context-aware suggestions for blocks, attributes, model names, strategies, scoring methods, and cross-references
- **Go-to-definition** — jump to prompt, skill, agent, and step definitions; open imported files
- **Diagnostics** — real-time validation via `agentspec validate` on save
- **Formatting** — auto-format on save via `agentspec fmt`
- **Plan preview** — show deployment plan in output panel

## Prerequisites

1. **Node.js 18+** and **npm**
2. **Go 1.25+** (to build the `agentspec` CLI)
3. **VS Code 1.85+**

## Build

```bash
cd vscode-agentspec

# Install dependencies
npm install

# Compile TypeScript
npm run compile
```

This produces the compiled extension in the `out/` directory.

## Load into VS Code

### Option 1: Development Host (recommended for development)

1. Open the `vscode-agentspec/` folder in VS Code
2. Press `F5` (or **Run > Start Debugging**)
3. A new VS Code window opens with the extension loaded
4. Open any `.ias` file to see it in action

To watch for changes during development:

```bash
npm run watch
```

Then press `F5` again — the extension reloads automatically when you save TypeScript files.

### Option 2: Install from VSIX (recommended for daily use)

```bash
# Install vsce if you don't have it
npm install -g @vscode/vsce

# Package the extension
cd vscode-agentspec
vsce package

# Install the .vsix file
code --install-extension vscode-agentspec-0.3.0.vsix
```

### Option 3: Symlink into extensions directory

```bash
# macOS / Linux
ln -s "$(pwd)/vscode-agentspec" ~/.vscode/extensions/vscode-agentspec

# Windows (PowerShell, run as admin)
New-Item -ItemType SymbolicLink -Path "$env:USERPROFILE\.vscode\extensions\vscode-agentspec" -Target "$(Get-Location)\vscode-agentspec"
```

Restart VS Code after symlinking.

## Build the AgentSpec CLI

The extension calls the `agentspec` CLI for validation, formatting, and plan preview. Build it from the repo root:

```bash
go build -o agentspec ./cmd/agentspec
```

Then either:
- Add the binary to your `PATH`, or
- Set the path in VS Code settings: **Settings > AgentSpec > Executable Path**

## Extension Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `agentspec.executablePath` | `agentspec` | Path to the `agentspec` CLI binary |
| `agentspec.formatOnSave` | `true` | Auto-format `.ias` files on save |
| `agentspec.validateOnSave` | `true` | Validate `.ias` files on save |

## Commands

Open the Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`) and type "AgentSpec":

| Command | Description |
|---------|-------------|
| **AgentSpec: Validate Current File** | Run `agentspec validate` on the active file |
| **AgentSpec: Format Current File** | Run `agentspec fmt` on the active file |
| **AgentSpec: Show Plan** | Run `agentspec plan` and display results |

## Snippets

Type any of these prefixes and press `Tab`:

| Prefix | Description |
|--------|-------------|
| `package` | Package declaration with version and lang |
| `import` | Import statement |
| `agent` | Agent block |
| `agent-config` | Agent with config, validation, and eval |
| `agent-ollama` | Ollama agent with local model selection |
| `prompt` | Prompt block |
| `skill` | Skill with input/output and tool |
| `skill-inline` | Skill with inline bash/python tool |
| `config` | Config block |
| `validate` | Validation rules block |
| `eval` | Eval test cases block |
| `on-input` | On input control flow block |
| `if` | If/else conditional |
| `foreach` | For each loop |
| `rule` | Validation rule |
| `case` | Eval test case |
| `deploy` | Deploy target |
| `pipeline` | Multi-agent pipeline |
| `tool-http` | HTTP tool |
| `tool-command` | Command tool |
| `tool-inline` | Inline code tool |
| `tool-mcp` | MCP tool |
| `memory` | Memory configuration |

## IntentLang 3.0 Support

This extension supports the full IntentLang 3.0 specification:

- **Package system** — `package`, `import ... as`
- **Core blocks** — `agent`, `prompt`, `skill`, `deploy`, `pipeline`
- **Agent blocks** — `config`, `validate`, `eval`, `on input`
- **Control flow** — `if` / `else if` / `else`, `for each ... in`
- **Actions** — `use skill`, `delegate to`, `respond`
- **Validation** — `rule` with `error` / `warning` severity and `when` conditions
- **Evaluation** — `case` with `input`, `expect`, `scoring`, `threshold`, `tags`
- **Config params** — typed parameters with `default`, `required`, `secret`
- **Multi-provider models** — `ollama/llama3.2`, `openai/gpt-4o`, `claude-sonnet-4-20250514`
- **Tool types** — `http`, `command`, `inline`, `mcp`
- **Scoring methods** — `contains`, `semantic`, `exact`, `regex`

## Project Structure

```
vscode-agentspec/
├── package.json                    # Extension manifest
├── tsconfig.json                   # TypeScript config
├── language-configuration.json     # Brackets, folding, indentation
├── syntaxes/
│   └── intentlang.tmLanguage.json  # TextMate grammar
├── snippets/
│   └── intentlang.json             # Code snippets
└── src/
    ├── extension.ts                # Entry point
    └── language-server/
        ├── completion.ts           # Autocomplete
        ├── definition.ts           # Go-to-definition
        └── diagnostics.ts          # Validation diagnostics
```
