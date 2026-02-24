# Secret Resolution

The secrets subsystem resolves secret references declared in IntentLang files to their actual values at runtime. It supports multiple providers and keeps sensitive values out of the IR and state file.

## Package

| Package | Path | Purpose |
|---------|------|---------|
| `secrets` | `internal/secrets/` | Resolver interface and provider implementations |

Source files:

| File | Purpose |
|------|---------|
| `resolver.go` | `Resolver` interface definition |
| `env.go` | Environment variable provider |
| `vault.go` | Vault-based secret provider |
| `redact.go` | Redaction utilities for log output |

## Resolver Interface

The core abstraction is the `Resolver` interface:

```go
type Resolver interface {
    Resolve(ctx context.Context, ref string) (string, error)
}
```

A resolver takes a reference string (the format depends on the provider) and returns the resolved secret value.

## Providers

### Environment Variable Provider

The `EnvResolver` reads secrets from environment variables. This is the default provider used by the runtime.

Reference format: the environment variable name (e.g., `OPENAI_API_KEY`).

The IntentLang secret declaration maps to an env lookup:

```text
secret "api-key" {
    env "OPENAI_API_KEY"
}
```

At resolve time:

1. The resolver looks up `OPENAI_API_KEY` in the process environment.
2. If the variable is set, its value is returned.
3. If unset, an error is returned.

### Store Provider (Vault)

The vault provider resolves secrets from an external secret store (e.g., HashiCorp Vault, AWS Secrets Manager):

```text
secret "db-password" {
    store "secrets/production/db-password"
}
```

At resolve time:

1. The resolver connects to the configured secret store.
2. It reads the value at the specified path.
3. The value is returned.

## Resolution Order

When the runtime encounters a secret reference, resolution follows this order:

1. **env provider** -- Check if the secret is declared with `env` source. Look up the environment variable.
2. **store provider** -- Check if the secret is declared with `store` source. Query the external secret store.
3. **Error** -- If no provider can resolve the reference, return an error.

## Usage in the Runtime

Secrets are resolved during runtime initialization, before tools are registered:

```go
func (rt *Runtime) registerTools(ctx context.Context, resolver secrets.Resolver) error {
    for _, skill := range rt.config.Skills {
        // ...
        if secs, ok := skill.Tool["secrets"].(map[string]interface{}); ok {
            for k, v := range secs {
                if ref, ok := v.(string); ok {
                    val, err := resolver.Resolve(ctx, ref)
                    if err != nil {
                        rt.logger.Warn("secret resolution failed", "key", k, "error", err)
                        continue
                    }
                    resolvedSecrets[k] = val
                }
            }
        }
        // Pass resolvedSecrets to tool executor
    }
}
```

The resolved values are passed to tool executors (command, inline) as environment variables or configuration, never stored in the state file.

## Security Properties

### Never Persisted

Secret values are resolved at runtime only. They never appear in:

- The IR document (`ir.Document`)
- The state file (`.agentspec.state.json`)
- Plan output
- Export artifacts

The IR and state only contain the secret *reference* (e.g., the environment variable name or store path), not the value.

### Redaction

The `redact.go` file provides utilities to mask secret values in log output:

- Values are replaced with `***REDACTED***` in structured log fields.
- Partial redaction shows only the last 4 characters for debugging: `***...abcd`.

### Runtime Scope

Resolved secrets are held in memory only for the duration of the runtime process. When the runtime shuts down, all resolved values are garbage collected.

## Tool Secret Injection

Different tool types receive secrets differently:

| Tool Type | How Secrets Are Injected |
|-----------|------------------------|
| `command` | Passed as environment variables to the subprocess |
| `inline` | Available as environment variables in the sandbox |
| `http` | Interpolated into headers or URL (via template variables) |
| `mcp` | Passed to the MCP server process as environment variables |

## Adding a New Secret Provider

To add a new provider (e.g., AWS Secrets Manager):

1. Create a new file in `internal/secrets/` (e.g., `aws.go`).

2. Implement the `Resolver` interface:

```go
type AWSResolver struct {
    client *secretsmanager.Client
    region string
}

func (r *AWSResolver) Resolve(ctx context.Context, ref string) (string, error) {
    output, err := r.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &ref,
    })
    if err != nil {
        return "", fmt.Errorf("aws secrets: %w", err)
    }
    return *output.SecretString, nil
}
```

3. Create a composite resolver that chains multiple providers:

```go
type ChainResolver struct {
    resolvers []Resolver
}

func (c *ChainResolver) Resolve(ctx context.Context, ref string) (string, error) {
    for _, r := range c.resolvers {
        val, err := r.Resolve(ctx, ref)
        if err == nil {
            return val, nil
        }
    }
    return "", fmt.Errorf("no resolver could handle ref %q", ref)
}
```

4. Update the runtime initialization to include the new provider in the chain.
