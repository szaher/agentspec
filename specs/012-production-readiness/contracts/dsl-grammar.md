# Contract: DSL Grammar Extensions

## New Blocks

### `user` block

```ebnf
user_block = "user" STRING "{" user_field* "}" ;
user_field = key_field | agents_field | role_field ;
key_field = "key" secret_ref ;
secret_ref = "secret" "(" STRING ")" ;
agents_field = "agents" "[" STRING ("," STRING)* "]" ;
role_field = "role" STRING ;
```

**Example**:
```
user "alice" {
  key secret("ALICE_API_KEY")
  agents ["support-agent", "search-agent"]
  role "invoke"
}
```

**Validation**:
- Secret reference must resolve to a declared `secret` block
- Agent names must resolve to declared `agent` blocks
- Role must be `"invoke"` or `"admin"` (default: `"invoke"`)

### `guardrail` block

```ebnf
guardrail_block = "guardrail" STRING "{" guardrail_field* "}" ;
guardrail_field = mode_field | keywords_field | patterns_field | fallback_field ;
mode_field = "mode" STRING ;
keywords_field = "keywords" "[" STRING ("," STRING)* "]" ;
patterns_field = "patterns" "[" STRING ("," STRING)* "]" ;
fallback_field = "fallback" STRING ;
```

**Example**:
```
guardrail "content-filter" {
  mode "block"
  keywords ["password", "SSN", "credit card"]
  patterns ["\\d{3}-\\d{2}-\\d{4}"]
  fallback "I cannot provide sensitive information."
}
```

**Validation**:
- Mode must be `"warn"` or `"block"`
- At least one of `keywords` or `patterns` must be present
- `fallback` is required when mode is `"block"`

## Agent Block Extensions

### `models` field (model fallback chain)

```ebnf
models_field = "models" "[" STRING ("," STRING)* "]" ;
```

**Semantics**: First model is primary. Subsequent models are tried in order on failure. Replaces the single `model` field when multiple models are needed.

**Validation**: At least one model required. Cannot use both `model` and `models` on the same agent.

### `budget` field

```ebnf
budget_field = "budget" ("daily" | "monthly") NUMBER ;
```

**Example**: `budget daily 10.0` — sets a $10/day spending limit.

**Validation**: Value must be positive. Period must be `"daily"` or `"monthly"`.

### `uses guardrail` reference

```ebnf
uses_guardrail = "uses" "guardrail" STRING ;
```

**Semantics**: Applies the referenced guardrail to this agent's output. Multiple guardrails can be applied (evaluated in order).

## IR Extensions

The IR gains the following new fields:

### Agent IR
- `Models []string` — fallback chain (replaces scalar `Model` when len > 1)
- `Guardrails []string` — references to guardrail definitions
- `BudgetDaily float64` — daily spending limit (0 = no limit)
- `BudgetMonthly float64` — monthly spending limit (0 = no limit)

### New IR top-level collections
- `Users []UserDef` — user definitions with key refs and agent access lists
- `Guardrails []GuardrailDef` — guardrail definitions with mode, keywords, patterns
