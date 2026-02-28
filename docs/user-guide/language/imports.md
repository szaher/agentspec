# Import

The `import` statement brings resources from other IntentLang files into the current file. This enables multi-file agent organization, code reuse, and modular project structure.

---

## Syntax

<!-- novalidate -->
```ias
import "<path>"
```

With an alias:

<!-- novalidate -->
```ias
import "<path>" as <alias>
```

Import statements appear at the top of the file, inside the `package` header block or immediately after it.

---

## Attributes

| Attribute | Required | Description                                                        |
|-----------|----------|--------------------------------------------------------------------|
| path      | Yes      | Relative file path to the imported `.ias` file (quoted string).     |
| `as`      | No       | Alias for the imported package. Used to qualify resource references. |

---

## Path Resolution

Import paths are resolved relative to the file that contains the `import` statement.

Given this directory structure:

```
my-agent/
  main.ias
  skills/
    search.ias
    respond.ias
  knowledge-base.ias
```

From `main.ias`:

<!-- novalidate -->
```ias
import "./skills/search.ias"
import "./skills/respond.ias"
import "./knowledge-base.ias" as kb
```

!!! note "Relative Paths Only"
    All import paths must start with `./` or `../`. Absolute paths are not supported.

---

## Aliased Imports

When an import uses the `as` keyword, the alias provides a namespace for referencing resources from the imported file. This prevents name collisions when importing resources with the same name from different files.

<!-- novalidate -->
```ias
import "./knowledge-base.ias" as kb
```

Resources from the aliased import can then be referenced using the `kb.` prefix in resource references.

---

## What Gets Imported

An import makes all top-level resources from the target file available in the importing file:

- Prompts
- Skills
- Agents
- Types
- Secrets
- Policies

The imported file must have its own `package` declaration. It does not need to share the same package name.

---

## Examples

### Multi-File Agent

Split skills into separate files for better organization.

**main.ias:**

<!-- novalidate -->
```ias
package "multi-file-agent" version "1.0.0" lang "3.0"

import "./skills/search.ias"
import "./skills/respond.ias"

prompt "system" {
  content "You are a helpful assistant that can search the web and format responses."
}

agent "assistant" {
  uses prompt "system"
  uses skill "web_search"
  uses skill "format_response"
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 5
}
```

**skills/search.ias:**

```ias
package "search-skills" version "1.0.0" lang "3.0"

skill "web_search" {
  description "Search the web for information"
  input {
    query string required
  }
  output {
    results string required
  }
  tool command {
    binary "echo"
    args "Search results for: {{query}}"
  }
}
```

**skills/respond.ias:**

```ias
package "respond-skills" version "1.0.0" lang "3.0"

skill "format_response" {
  description "Format a response for the user"
  input {
    content string required
  }
  output {
    formatted string required
  }
  tool command {
    binary "echo"
    args "Formatted: {{content}}"
  }
}
```

### Aliased Import

Use an alias when importing shared resources:

<!-- novalidate -->
```ias
package "validated-agent" version "1.0.0" lang "3.0"

import "./knowledge-base.ias" as kb

prompt "support-system" {
  content "You are a helpful customer support agent. Use the knowledge base to answer questions accurately."
}

agent "support-agent" {
  uses prompt "support-system"
  uses skill "knowledge-search"
  model "claude-sonnet-4-5-20250514"
  strategy "react"
  max_turns 10
}
```

---

## Best Practices

- **Group related resources** -- Put skills that belong together in the same file (e.g., `skills/search.ias`).
- **Use aliases** -- When importing files that might have name conflicts, use `as` to namespace them.
- **Keep the main file lean** -- Define agents and prompts in the main file; move skills and types to imported files.
- **One agent per file** -- For complex projects, consider putting each agent in its own file with its prompt.

---

## See Also

- [Agent](agent.md) -- Agents that reference imported resources
- [Skill](skill.md) -- Skills commonly organized in imported files
- [Prompt](prompt.md) -- Prompts that can be shared across files
