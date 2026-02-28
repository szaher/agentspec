# Type

A **type** defines a reusable, named data structure for use in skill input/output
schemas and other typed contexts. IntentLang supports three type variants: struct,
enum, and list.

---

## Syntax

IntentLang provides three ways to define a type.

### Struct

A struct type defines a record with named, typed fields.

```ias
type "<name>" {
  <field-name> <field-type> [required] [default "<value>"]
  # ... more fields
}
```

### Enum

An enum type defines a fixed set of allowed string values.

```ias
type "<name>" enum ["<value1>", "<value2>", ...]
```

### List

A list type defines an ordered collection of elements of a single type.

```ias
type "<name>" list <element-type>
```

---

## Supported Field Types

The following scalar types are available for struct fields and list element types:

| Type     | Description                          | Example Values              |
|:---------|:-------------------------------------|:----------------------------|
| `string` | UTF-8 text                           | `"hello"`, `""`             |
| `int`    | Signed 64-bit integer                | `0`, `42`, `-1`             |
| `bool`   | Boolean                              | `true`, `false`             |
| `float`  | 64-bit floating-point number         | `3.14`, `-0.5`, `1.0`      |

!!! info "Type references"
    A struct field can also reference another named type by its identifier.
    This enables composition of complex data structures.

---

## Struct Field Modifiers

| Modifier            | Description                                                   |
|:--------------------|:--------------------------------------------------------------|
| `required`          | The field must be provided. Validation fails if it is absent. |
| `default "<value>"` | The default value used when the field is not provided.        |

!!! warning "Mutual exclusivity"
    A field cannot be both `required` and have a `default` value. If a field
    has a default, it is implicitly optional. The validator rejects definitions
    that specify both.

---

## Rules

- Type names must be **unique within the package**.
- Struct types must contain **at least one field**.
- Enum types must contain **at least one value**.
- Enum values must be **unique** within the enum.
- List types require exactly one element type.
- Field names within a struct must be unique.

---

## Examples

### Struct Type

Define a structured input schema for a customer lookup skill.

```ias
package "types-demo" version "0.1.0" lang "2.0"

type "customer-query" {
  customer_id string required
  include_history bool default "false"
  max_results int default "10"
}

type "address" {
  street string required
  city string required
  state string required
  zip string required
  country string default "US"
}

skill "lookup-customer" {
  description "Look up a customer by ID"
  input  { query customer-query required }
  output { result string }
  tool command { binary "customer-lookup" }
}

prompt "support" {
  content "You are a customer support assistant."
}

agent "support-bot" {
  uses prompt "support"
  uses skill "lookup-customer"
  model "claude-sonnet-4-20250514"
}

deploy "local" target "process" {
  default true
}
```

### Enum Type

Restrict a field to a known set of values.

```ias
type "priority" enum ["low", "medium", "high", "critical"]

type "ticket-status" enum ["open", "in_progress", "resolved", "closed"]
```

Enum types are useful for fields that accept only predefined options. The
validator rejects any value not in the enum list.

### List Type

Define a collection type for repeated elements.

```ias
type "tags" list string

type "scores" list float

type "ids" list int
```

### Combining Types

Types can be composed to build richer schemas.

```ias
type "severity" enum ["low", "medium", "high", "critical"]

type "finding" {
  title string required
  description string required
  severity severity required
  line_number int default "0"
  fixable bool default "true"
}

type "findings" list finding
```

In this example, `finding` references the `severity` enum, and `findings` is a
list of `finding` structs. This composition allows skill schemas to express
complex, nested data shapes.

!!! tip "When to define custom types"
    If the same field structure appears in more than one skill's input or output
    block, extract it into a named type. This reduces duplication and keeps
    schemas consistent across the package.

---

## See Also

- [Skill](skill.md) -- uses types in input/output schemas
- [Agent](agent.md) -- the resource type that invokes skills
