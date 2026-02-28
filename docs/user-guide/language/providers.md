# Providers

IntentLang supports multiple LLM providers through the `model` attribute on agents. The provider is determined from the model string, either by an explicit prefix or by auto-detection from the model name.

---

## Syntax

The `model` attribute in an agent block accepts two formats:

**Explicit provider prefix:**

<!-- novalidate -->
```ias
model "<provider>/<model-name>"
```

**Model name only (auto-detected):**

<!-- novalidate -->
```ias
model "<model-name>"
```

---

## Supported Providers

| Provider    | Prefix       | Example Models                                    |
|-------------|-------------|---------------------------------------------------|
| Anthropic   | `anthropic/` | `claude-sonnet-4-20250514`, `claude-sonnet-4-5-20250514` |
| OpenAI      | `openai/`    | `gpt-4o`, `o1`, `o3`, `o4`                        |
| Ollama      | `ollama/`    | `llama3.1`, `llama3.2`, `mistral`, `codellama`     |

---

## Auto-Detection

When no provider prefix is specified, the provider is inferred from the model name:

| Pattern                  | Detected Provider |
|--------------------------|-------------------|
| Starts with `claude`     | Anthropic         |
| Starts with `gpt-`, `o1`, `o3`, or `o4` | OpenAI |
| `OLLAMA_HOST` env var is set | Ollama         |
| `OPENAI_API_KEY` env var is set | OpenAI      |
| None of the above        | Anthropic (default) |

**Examples:**

<!-- novalidate -->
```ias
model "claude-sonnet-4-20250514"   # Detected as Anthropic
model "gpt-4o"                     # Detected as OpenAI
model "llama3.1"                   # Detected based on env vars, or falls back to Anthropic
```

!!! tip "Use Explicit Prefixes"
    For clarity and to avoid ambiguity, prefer the explicit `provider/model` format, especially for models whose names do not match a known pattern (e.g., `ollama/llama3.1` instead of just `llama3.1`).

---

## Environment Variables

Each provider uses specific environment variables for configuration.

### Anthropic

| Variable           | Description                              |
|--------------------|------------------------------------------|
| `ANTHROPIC_API_KEY` | API key for Anthropic. Read by the SDK automatically. |

### OpenAI

| Variable          | Description                                              |
|-------------------|----------------------------------------------------------|
| `OPENAI_API_KEY`  | API key for OpenAI.                                       |
| `OPENAI_BASE_URL` | Custom base URL for OpenAI-compatible APIs (e.g., Azure). |

### Ollama

| Variable      | Description                                                      |
|---------------|------------------------------------------------------------------|
| `OLLAMA_HOST` | Ollama server address. Defaults to `http://localhost:11434`.      |

!!! note "Ollama Requires No API Key"
    Ollama runs locally and does not require an API key. Just ensure the Ollama server is running and `OLLAMA_HOST` is set if it is not on the default address.

---

## Using with Local Models (Ollama)

Ollama enables running open-source models locally. To use Ollama with AgentSpec:

1. Install and start Ollama:

    ```bash
    ollama serve
    ```

2. Pull a model:

    ```bash
    ollama pull llama3.1
    ```

3. Reference it in your agent:

    <!-- novalidate -->
    ```ias
    agent "local-assistant" {
      uses prompt "system"
      model "ollama/llama3.1"
      strategy "react"
      max_turns 10
      stream true
    }
    ```

4. Run the agent:

    ```bash
    agentspec apply agent.ias
    ```

If the Ollama server is on a non-default address, set the environment variable:

```bash
export OLLAMA_HOST=http://192.168.1.100:11434
```

---

## OpenAI-Compatible APIs

The `OPENAI_BASE_URL` environment variable allows connecting to any OpenAI-compatible API endpoint. This is useful for services like Azure OpenAI, Together AI, or self-hosted inference servers.

```bash
export OPENAI_API_KEY=your-key
export OPENAI_BASE_URL=https://your-endpoint.example.com/v1
```

Then use the `openai/` prefix with any model name the endpoint supports:

<!-- novalidate -->
```ias
model "openai/your-custom-model"
```

---

## Examples

### Anthropic (Default)

<!-- novalidate -->
```ias
agent "assistant" {
  uses prompt "system"
  model "claude-sonnet-4-20250514"
  strategy "react"
}
```

### OpenAI

<!-- novalidate -->
```ias
agent "assistant" {
  uses prompt "system"
  model "openai/gpt-4o"
  strategy "react"
}
```

### Ollama (Local)

<!-- novalidate -->
```ias
agent "coder" {
  uses prompt "coder-system"
  uses skill "list-files"
  uses skill "read-file"
  uses skill "run-command"
  model "ollama/llama3.1"
  strategy "react"
  max_turns 10
  stream true
}
```

### Multiple Providers in One Project

<!-- novalidate -->
```ias
agent "fast-router" {
  uses prompt "router-system"
  model "ollama/llama3.1"
  strategy "router"
}

agent "quality-writer" {
  uses prompt "writer-system"
  model "claude-sonnet-4-5-20250514"
  strategy "reflexion"
}

agent "code-generator" {
  uses prompt "coder-system"
  model "openai/gpt-4o"
  strategy "react"
}
```

---

## See Also

- [Agent](agent.md) -- The `model` attribute on agents
- [Config](config.md) -- Runtime configuration for API keys and endpoints
