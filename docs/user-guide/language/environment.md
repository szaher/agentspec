# Environment

An **environment** block defines an overlay that overrides resource attributes for a
specific deployment context. Environments let you maintain a single `.ias` source
file while varying configuration across development, staging, production, or any
custom environment.

---

## Syntax

<!-- novalidate -->
```ias
environment "<name>" {
  agent "<agent-name>" {
    <attribute> <value>
  }
  # ... more agent overrides
}
```

`<name>` is a unique identifier for the environment (e.g., `"dev"`, `"staging"`,
`"prod"`). Inside the environment block, you nest agent blocks that specify which
attributes to override.

---

## Attributes

The environment block itself has no direct attributes beyond its name. It contains
**agent override blocks**, each of which can set any agent-level attribute.

### Agent Override Attributes

| Attribute | Type   | Description                                             |
|:----------|:-------|:--------------------------------------------------------|
| `model`   | string | Override the model used by the agent in this environment. |

!!! info "Extensibility"
    The override mechanism supports any attribute that is valid on the target
    resource type. As IntentLang evolves, additional resource types and
    attributes may become overridable.

---

## Rules

- Environment names must be **unique within the package**.
- Each agent reference inside an environment must correspond to an agent defined in the same package.
- Overrides are applied **on top of** the base resource definition -- attributes not mentioned in the override retain their original values.
- Multiple environments can override the same agent with different values.
- An environment block must contain **at least one** override.

!!! warning "Non-existent agent references"
    `agentspec validate` rejects environment blocks that reference agents not
    defined in the package. Always ensure the agent name matches exactly.

---

## How Overrides Work

Consider an agent defined with a base model:

<!-- novalidate -->
```ias
agent "assistant" {
  uses prompt "greeting"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}
```

An environment overlay can change the model without duplicating the rest of the
definition:

<!-- novalidate -->
```ias
environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}
```

When applied with `--env dev`, the agent runs with `claude-haiku-latest`. All
other attributes (`uses prompt`, `uses skill`) remain unchanged.

---

## Examples

### Dev and Prod Model Overrides

The most common use case: use a cheaper, faster model in development and a more
capable model in production.

```ias
package "multi-env" version "0.1.0" lang "2.0"

prompt "greeting" {
  content "You are a helpful assistant."
}

skill "search" {
  description "Search the web"
  input  { query string required }
  output { results string }
  tool command { binary "search-tool" }
}

agent "assistant" {
  uses prompt "greeting"
  uses skill "search"
  model "claude-sonnet-4-20250514"
}

environment "dev" {
  agent "assistant" {
    model "claude-haiku-latest"
  }
}

environment "staging" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
  }
}

environment "prod" {
  agent "assistant" {
    model "claude-sonnet-4-20250514"
  }
}

deploy "local" target "process" {
  default true
}
```

!!! tip "Cost management"
    Using a lighter model like `claude-haiku-latest` in development reduces
    costs during iterative testing while preserving the full agent definition.
    Switch to a more capable model only for staging and production.

### Multiple Agent Overrides

Override different agents within the same environment.

<!-- novalidate -->
```ias
agent "code-analyzer" {
  uses prompt "analyzer"
  uses skill "analyze-code"
  model "claude-sonnet-4-20250514"
}

agent "review-summarizer" {
  uses prompt "summarizer"
  uses skill "post-review"
  model "claude-sonnet-4-20250514"
}

environment "dev" {
  agent "code-analyzer" {
    model "claude-haiku-latest"
  }
  agent "review-summarizer" {
    model "claude-haiku-latest"
  }
}

environment "prod" {
  agent "code-analyzer" {
    model "claude-sonnet-4-20250514"
  }
  agent "review-summarizer" {
    model "claude-sonnet-4-20250514"
  }
}
```

Each agent can be overridden independently. In `dev`, both agents use the
lighter model; in `prod`, both use the full model.

---

## CLI Usage

Use the `--env` flag to select which environment overlay to apply:

```bash
# Apply with dev environment overrides
agentspec apply my-agent.ias --env dev

# Apply with production environment overrides
agentspec apply my-agent.ias --env prod

# Plan with a specific environment to preview changes
agentspec plan my-agent.ias --env staging
```

When no `--env` flag is provided, the base resource definitions are used without
any overrides.

!!! info "Validation per environment"
    You can validate a specific environment configuration:

    ```bash
    agentspec validate my-agent.ias --env prod
    ```

    This checks that all overrides are valid and that the resulting
    configuration passes all validation rules, including any active policies.

---

## Environments and Policies

Environment overrides interact with [policy](policy.md) blocks. A policy that
denies a specific model will reject an environment that overrides an agent to use
that model.

<!-- novalidate -->
```ias
policy "no-haiku-in-prod" {
  deny model claude-haiku-latest
}

environment "prod" {
  agent "assistant" {
    model "claude-haiku-latest"   # Rejected by policy
  }
}
```

Running `agentspec validate --env prod` would fail because the policy prohibits
`claude-haiku-latest`.

!!! warning "Policy enforcement"
    Policies are evaluated **after** environment overrides are applied. An
    override that satisfies the base definition may still be rejected by a
    policy. Always validate each environment you intend to deploy.

---

## See Also

- [Agent](agent.md) -- the resource type that environments override
- [Secret](secret.md) -- sensitive values that may vary by environment
- [Pipeline](pipeline.md) -- pipelines execute with the active environment's agent configuration
