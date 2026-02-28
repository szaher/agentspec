# policy

The `policy` block defines security and governance constraints that are enforced during validation and deployment. Policies let you prohibit certain configurations, mandate the presence of secrets, and explicitly permit resources that might otherwise be restricted.

---

## Syntax

```ias
policy "<name>" {
  deny <resource-type> <resource-name>
  require <resource-type> <resource-name>
  allow <resource-type> <resource-name>
}
```

The policy name must be unique within the package. Each policy block contains one or more **rules**, where each rule is an action followed by a resource type and resource name.

---

## Actions

### deny

Prohibits the use of a specific resource. If any agent or configuration references a denied resource, validation fails with an error.

```ias
deny model claude-haiku-latest
deny skill unsafe-execute
```

### require

Mandates that a specific resource is defined in the package. Validation fails if the required resource does not exist.

```ias
require secret api-key
require secret db-connection
```

### allow

Explicitly permits a resource. This is useful in combination with broad deny rules or when documenting approved resources for compliance purposes.

```ias
allow model claude-sonnet-4-20250514
allow skill web-search
```

---

## Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `deny` | rule | Prohibit the use of a resource. Validation fails if the resource is referenced. |
| `require` | rule | Mandate that a resource exists. Validation fails if the resource is missing. |
| `allow` | rule | Explicitly permit a resource. |

### Supported Resource Types

| Resource Type | Description | Example |
|---------------|-------------|---------|
| `model` | LLM model identifier | `deny model claude-haiku-latest` |
| `skill` | Skill name | `deny skill unsafe-execute` |
| `secret` | Secret name | `require secret api-key` |

---

## Validation Behavior

Policies are evaluated during `agentspec validate` and `agentspec plan`. When a policy rule is violated, the toolchain reports a clear error indicating the policy name, the violated rule, and the offending resource.

- **deny** rules cause a validation error if any agent in the package uses the denied resource.
- **require** rules cause a validation error if the specified resource is not declared anywhere in the package.
- **allow** rules do not cause errors on their own; they serve as explicit permits.

!!! warning "Policies are package-scoped"
    Policy rules apply to the entire package. A `deny model` rule prevents **all** agents in the package from using that model, not just a single agent.

---

## Examples

### Production Safety Policy

This policy ensures that production deployments use high-quality models and that all required secrets are available.

```ias
package "data-pipeline" version "0.1.0" lang "2.0"

prompt "etl" {
  content "You are a data engineering assistant. Help users extract
           data from sources, transform it according to rules, and
           load it into target systems. Always validate data quality
           before loading."
}

skill "extract" {
  description "Extract data from a source system"
  input {
    source string required
  }
  output {
    raw_data string
  }
  tool command {
    binary "data-extract"
  }
}

skill "load" {
  description "Load data into the target database"
  input {
    data string required
  }
  output {
    rows_inserted string
  }
  tool command {
    binary "data-load"
  }
}

agent "etl-bot" {
  uses prompt "etl"
  uses skill "extract"
  uses skill "load"
  model "claude-sonnet-4-20250514"
}

secret "db-connection" {
  env(DATABASE_URL)
}

secret "source-api-key" {
  env(SOURCE_API_KEY)
}

policy "production-safety" {
  deny model claude-haiku-latest
  require secret db-connection
  require secret source-api-key
}

deploy "local" target "process" {
  default true
}
```

!!! tip "Combine policies with environments"
    Use policies alongside `environment` blocks to enforce different constraints per deployment stage. For example, you might allow `claude-haiku-latest` in a `dev` environment but deny it in production via a policy.

### Minimal Policy

A simple policy that ensures a single secret is always present:

```ias
policy "api-access" {
  require secret api-key
}
```

### Multi-Rule Policy

Policies can combine multiple actions to express complex constraints:

```ias
policy "compliance" {
  deny model claude-haiku-latest
  deny skill raw-sql-execute
  require secret audit-log-key
  allow model claude-sonnet-4-20250514
}
```

---

## See Also

- [secret](secret.md) -- Declaring secrets that policies can require
- [environment](environment.md) -- Environment overlays for per-stage configuration
- [agent](agent.md) -- Agent blocks where model and skill references are defined
