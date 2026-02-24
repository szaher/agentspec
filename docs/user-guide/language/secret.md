# Secret

A **secret** block defines a reference to a sensitive value such as an API key,
database credential, or authentication token. Secrets are never stored in the
`.ias` source file itself -- they are resolved at runtime from an external source.

IntentLang supports two secret sources: **environment variables** and a **secure
store**.

---

## Syntax

### From Environment Variable

```ias novalidate
secret "<name>" {
  env(<VARIABLE_NAME>)
}
```

Resolves the secret value from the environment variable `VARIABLE_NAME` at
runtime.

### From Secure Store

```ias novalidate
secret "<name>" {
  store(<path/to/secret>)
}
```

Resolves the secret value from a secure store (such as HashiCorp Vault, AWS
Secrets Manager, or a local keychain) using the given path.

---

## Attributes

| Attribute | Type   | Required | Description                                                         |
|:----------|:-------|:---------|:--------------------------------------------------------------------|
| `env`     | string | Conditional | Environment variable name. Required if `store` is not used.     |
| `store`   | string | Conditional | Path in the secure store. Required if `env` is not used.        |

!!! warning "Exactly one source"
    Each secret block must specify exactly one source -- either `env` or `store`.
    Specifying both, or neither, is a validation error.

---

## Rules

- Secret names must be **unique within the package**.
- A secret block must contain exactly **one** source directive (`env` or `store`).
- Environment variable names follow shell conventions: uppercase letters, digits, and underscores.
- Store paths use forward slashes as separators and must not be empty.
- Secret values are resolved at **apply time**, not at validate or plan time.

!!! info "Validation behavior"
    `agentspec validate` checks that secrets are syntactically correct and that
    all references to them are valid. It does **not** check whether the
    environment variable is set or the store path exists -- that happens at
    `agentspec apply` time.

---

## Referencing Secrets

Secrets are referenced by other blocks using the `secret "<name>"` syntax.

### In Server Auth

```ias fragment
secret "api-token" {
  env(MCP_API_TOKEN)
}

server "remote-server" {
  transport "sse"
  url "https://mcp.example.com/api"
  auth "api-token"
}
```

### In Policy Blocks

```ias novalidate
secret "db-password" {
  store(production/database/password)
}

policy "require-db-creds" {
  require secret db-password
}
```

### In Deploy Blocks

```ias fragment
secret "registry-key" {
  env(DOCKER_REGISTRY_KEY)
}

deploy "production" target "kubernetes" {
  namespace "agents"
  replicas 3
}
```

---

## Examples

### Environment Variable Secrets

The most common pattern for local development and CI/CD pipelines.

```ias
package "secret-demo" version "0.1.0" lang "2.0"

prompt "assistant" {
  content "You are a helpful assistant."
}

skill "search" {
  description "Search the web"
  input  { query string required }
  output { results string }
  tool command { binary "search-tool" }
}

agent "search-bot" {
  uses prompt "assistant"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}

secret "search-api-key" {
  env(SEARCH_API_KEY)
}

secret "openai-key" {
  env(OPENAI_API_KEY)
}

deploy "local" target "process" {
  default true
}
```

Before running `agentspec apply`, set the environment variables:

```bash
export SEARCH_API_KEY="sk-..."
export OPENAI_API_KEY="sk-..."
agentspec apply secret-demo.ias
```

### Secure Store Secrets

For production environments where secrets are managed centrally.

```ias novalidate
secret "db-connection" {
  store(production/database/connection-string)
}

secret "api-signing-key" {
  store(production/api/signing-key)
}

secret "tls-certificate" {
  store(production/tls/cert)
}
```

!!! tip "Store path conventions"
    Organize store paths by environment and service:
    `<environment>/<service>/<secret-name>`. This makes it easy to manage
    secrets across dev, staging, and production environments.

### Combining Both Sources

A package can mix `env` and `store` secrets. This is common when some secrets
are readily available as environment variables (e.g., in CI) while others must
come from a centralized store.

```ias novalidate
# CI-friendly: set in the pipeline environment
secret "github-token" {
  env(GITHUB_TOKEN)
}

# Production: managed in Vault
secret "database-url" {
  store(production/postgres/url)
}

policy "production-safety" {
  require secret database-url
}
```

---

## See Also

- [Server](server.md) -- uses secrets for MCP server authentication
- [Environment](environment.md) -- override configurations per environment
