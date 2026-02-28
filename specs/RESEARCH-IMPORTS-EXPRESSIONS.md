# IntentLang DSL Enhancement Research

Research conducted: 2026-02-28

## Executive Summary

This document provides comprehensive research on two key topics for IntentLang DSL enhancement:

1. **Import/Module Systems** - How declarative languages handle imports, version resolution, and circular dependency detection
2. **Sandboxed Expression Evaluation** - Lightweight, safe expression evaluators for runtime control flow in compiled agents

## Topic 1: Import/Module Systems for DSLs

### 1.1 Go Modules

**Import Syntax:**
```go
import (
    "fmt"                           // Standard library
    "github.com/user/repo/package"  // External package
)
```

**Resolution Algorithm:**

Go uses **Minimal Version Selection (MVS)**, which is fundamentally different from traditional dependency resolution:

- Each package specifies a minimum version of each dependency
- The go command selects the semantically highest of the versions explicitly listed by require directives
- Prefers minimum allowed versions (the exact version the author used) rather than latest versions
- No SAT solvers required, producing deterministic builds without lock files

**Version Pinning Mechanism:**

```go
// go.mod file
module github.com/user/project

go 1.25

require (
    github.com/pkg/errors v0.9.1
    github.com/sirupsen/logrus v1.8.1
)
```

- Versions follow semantic versioning (vX.Y.Z)
- Major version changes (v2+) are part of the import path: `github.com/user/repo/v2`
- Automatic upgrades via `go get -u` or manual edits to go.mod
- go.sum file stores cryptographic checksums for verification

**Circular Dependency Detection:**

- Go compiler detects import cycles at compile time
- Error: "import cycle not allowed"
- Uses simple depth-first search during import resolution

**Package Registry Pattern:**

- **GOPROXY protocol**: HTTP-based proxy service
- Default: `GOPROXY=proxy.golang.org,direct`
- Module proxy endpoints:
  - `GET /{module}/@v/list` - list available versions
  - `GET /{module}/@v/{version}.info` - version metadata
  - `GET /{module}/@v/{version}.mod` - go.mod file
  - `GET /{module}/@v/{version}.zip` - module source archive
- Companion variables: GOPRIVATE, GONOPROXY for private modules

**Strengths:**
- Deterministic, reproducible builds without lock files
- Simple MVS algorithm is easy to understand and predict
- Built-in module proxy protocol with caching
- Strong tooling support (go get, go mod tidy, go mod graph)

**Weaknesses:**
- Major version changes require import path changes
- Can result in multiple versions of same major version in dependency tree

**Sources:**
- [Minimal Version Selection](https://research.swtch.com/vgo-mvs)
- [The Principles of Versioning in Go](https://research.swtch.com/vgo-principles)
- [Go Modules Reference](https://go.dev/ref/mod)
- [Go Module Proxy](https://proxy.golang.org/)
- [Go proxy for GitLab](https://docs.gitlab.com/user/packages/go_proxy/)

---

### 1.2 Terraform Modules

**Import Syntax:**

```hcl
# Local path
module "vpc" {
  source = "./modules/vpc"
}

# Registry module
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.14.0"
}

# Git repository
module "vpc" {
  source = "git::https://github.com/user/repo.git//modules/vpc?ref=v1.2.0"
}

# Private registry
module "vpc" {
  source  = "app.terraform.io/example_corp/vpc/aws"
  version = "0.9.3"
}
```

**Resolution Algorithm:**

- Local paths: resolved relative to the calling module
- Registry modules: fetched from Terraform Registry or private registries
- Git sources: support branches, tags, and commit SHAs via `?ref=` parameter
- Supports shallow clones via `?depth=` parameter
- No automatic dependency resolution between modules - each module is independent

**Version Pinning Mechanism:**

```hcl
module "example" {
  source  = "hashicorp/consul/aws"
  version = "~> 0.1"  # Pessimistic constraint
}
```

Version constraints:
- `= 1.2.0` - exact version
- `>= 1.2.0` - any version >= 1.2.0
- `~> 1.2.0` - any version >= 1.2.0 and < 1.3.0
- `>= 1.0, < 2.0` - range constraint

**Lock File: `.terraform.lock.hcl`**

- Introduced in Terraform 0.14
- **Important limitation**: Locks only provider versions, NOT module versions
- Modules always select newest available version matching constraints
- Workaround: use exact version constraints for repeatability

```hcl
provider "registry.terraform.io/hashicorp/aws" {
  version     = "4.45.0"
  constraints = "~> 4.0"
  hashes = [
    "h1:abc123...",
  ]
}
```

**Circular Dependency Detection:**

- Terraform detects circular dependencies during the plan phase
- Error: "Cycle: module.a -> module.b -> module.a"
- Uses graph traversal to build dependency tree
- Module dependencies are explicit via variable references

**Package Registry Pattern:**

- Terraform Registry (registry.terraform.io)
- Private registry protocol: discovery document at `/.well-known/terraform.json`
- Registry API endpoints:
  - `GET /{namespace}/{name}/{provider}/versions` - list versions
  - `GET /{namespace}/{name}/{provider}/{version}/download` - download URL
- No central caching proxy like Go (each terraform init downloads directly)

**Strengths:**
- Flexible source options (local, git, registry, HTTP)
- Clear version constraint syntax
- Works well for infrastructure composition
- Supports private registries

**Weaknesses:**
- Modules not tracked in lock file (only providers)
- No transitive dependency resolution
- Version drift possible without exact constraints
- No built-in circular dependency prevention at authoring time

**Sources:**
- [Use modules in your configuration](https://developer.hashicorp.com/terraform/language/modules/configuration)
- [Module Sources](https://developer.hashicorp.com/terraform/language/modules/sources)
- [Dependency Lock File](https://developer.hashicorp.com/terraform/language/files/dependency-lock)
- [Terraform Lock Files Explained](https://spacelift.io/blog/terraform-lock-hcl-file)

---

### 1.3 Protobuf Imports

**Import Syntax:**

```protobuf
syntax = "proto3";

import "google/protobuf/timestamp.proto";
import "common/user.proto";
import public "common/shared.proto";  // Re-exported
```

**Resolution Algorithm:**

1. Compiler searches for imported files in directories specified by `--proto_path` (or `-I`) flags
2. Import paths are resolved relative to these proto_path directories
3. If multiple proto_path directories are specified, they are searched in order
4. Modern approach (Buf): workspace-based with explicit dependency management

**Traditional protoc resolution:**
```bash
protoc --proto_path=. --proto_path=./vendor \
       --go_out=. myproto.proto
```

**Buf resolution (modern):**
```yaml
# buf.yaml
version: v1
deps:
  - buf.build/googleapis/googleapis
  - buf.build/envoyproxy/protoc-gen-validate
```

**Version Pinning Mechanism:**

Traditional protoc: No built-in versioning, relies on file organization and manual versioning

Buf (modern approach):
```yaml
# buf.lock
version: v1
deps:
  - remote: buf.build
    owner: googleapis
    repository: googleapis
    commit: 62f35d8aed1149c291d606d958a7ce32
```

**Circular Dependency Detection:**

- protoc detects circular imports at compile time
- Error: "import X.proto transitively imports Y.proto which imports X.proto"
- Uses stack-based detection during recursive import traversal
- Public imports create a new dependency edge (transitive)

**Package Registry Pattern:**

**Buf Schema Registry (BSR):**
- Central registry at buf.build
- Module naming: `buf.build/{owner}/{repository}`
- Git-like commit-based versioning (not semver by default, but supports tags)
- Built-in dependency management via buf.yaml
- Private BSR instances supported for enterprise

**Type Name Resolution:**
- First searches innermost scope, then progressively outer scopes
- Each package is "inner" to its parent package
- Leading dot (`.foo.bar.Baz`) starts from outermost scope
- Similar to C++ namespace resolution

**Strengths:**
- Simple path-based resolution
- Clear scope rules for type names
- Modern tooling (Buf) adds proper dependency management
- Public imports enable API evolution

**Weaknesses:**
- Traditional protoc has no version management
- Import paths can be brittle without proper tooling
- No automatic dependency fetching in protoc

**Sources:**
- [Language Guide (proto 3)](https://protobuf.dev/programming-guides/proto3/)
- [Manage dependencies - Buf Docs](https://buf.build/docs/bsr/module/dependency-management/)
- [Protocol Buffers Language Specification](https://protobuf.dev/reference/protobuf/proto3-spec/)

---

### 1.4 Dhall, CUE, Jsonnet

#### Dhall

**Import Syntax:**

```dhall
-- Local file
let config = ./config.dhall

-- Remote import (HTTPS)
let types = https://prelude.dhall-lang.org/v20.0.0/package.dhall

-- Environment variable
let apiKey = env:API_KEY as Text
```

**Resolution Algorithm:**

- Local paths: relative to importing file (./file.dhall, ../parent.dhall)
- Remote imports: HTTPS URLs fetched at evaluation time
- Environment variables: resolved at runtime
- Referential transparency: remote imports cannot depend on local imports
- All imports are cryptographically hashed for integrity

**Version Pinning Mechanism:**

```dhall
-- Semantic versioning in URL path
let Prelude = https://prelude.dhall-lang.org/v21.1.0/package.dhall

-- Integrity checks (semantic hash)
let example =
      https://example.com/config.dhall
        sha256:abc123...
```

- Versions typically encoded in URL path
- Semantic hashing ensures content integrity
- Import resolution is cached based on hash

**Circular Dependency Detection:**

- Dhall prohibits recursive imports
- All imports must form a directed acyclic graph (DAG)
- Detected at evaluation time with clear error messages

**Security Model:**

- **CORS protection**: Transitive imports must opt-in via CORS headers
- **Referential transparency**: Remote imports cannot access local files
- **No arbitrary side effects**: Only imports allowed (no file writes, network calls)
- **Protection against SSRF**: Built-in CORS checks prevent server-side request forgery

**Package Registry:**

- No central registry
- Convention: version-tagged HTTPS URLs
- Common practice: GitHub releases with semantic versioning in path
- Decentralized model with cryptographic verification

**Strengths:**
- Strong security model with CORS and referential transparency
- Cryptographic integrity checking
- Safe for untrusted code execution
- Functional purity prevents side effects

**Weaknesses:**
- No central package registry
- Can be verbose and difficult to read
- Performance issues (large templates can take minutes and gigabytes of RAM)
- Network dependency for remote imports

**Sources:**
- [Safety Guarantees](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)
- [The Dhall configuration language](https://dhall-lang.org/)
- [Safety guarantees](https://github.com/dhall-lang/dhall-lang/blob/master/docs/discussions/Safety-guarantees.md)

---

#### CUE

**Import Syntax:**

```cue
import "encoding/json"                    // Standard library
import "github.com/owner/repo/pkg"        // User package
import alias "github.com/other/pkg"       // Aliased import
```

**Resolution Algorithm:**

- Imports without domain assumed to be built-in standard library
- Domain-based imports (github.com/owner/repo) follow Go-style conventions
- **No relative imports** (security decision - only absolute imports allowed)
- Module system modeled after Go modules
- Package resolution via cue.mod directory structure

**Version Pinning Mechanism:**

Proposed (in development):
```
// cue.mod/module.cue
module: "github.com/user/project"
language: version: "v0.5.0"

require: {
    "github.com/other/pkg": "v1.2.3"
}
```

- Planned: Minimal Version Selection (like Go)
- Planned: OCI-compliant artifact registries
- Currently: mostly local development, limited remote package support

**Circular Dependency Detection:**

- CUE detects cycles during unification
- Cycles in data structures allowed (via comprehensions)
- Import cycles prevented by DAG requirement

**Package Registry:**

- **Planned**: OCI registry protocol (e.g., Docker registries)
- **Current**: Limited support, mostly local packages
- URI-style package naming for future registry integration

**Strengths:**
- Security via absolute-only imports
- Go-style module system (familiar to Go developers)
- Strong typing and validation built-in
- Unique composability model (unification, not inheritance)

**Weaknesses:**
- Package management still in early stages
- Limited remote package ecosystem
- No relative imports can be inconvenient for local development

**Sources:**
- [Modules, Packages, and Instances](https://cuelang.org/docs/concept/modules-packages-instances/)
- [Imports](https://cuelang.org/docs/tour/packages/imports/)
- [Proposal: package management](https://github.com/cue-lang/cue/issues/851)
- [CUE Modules](https://cuelang.org/docs/reference/modules/)

---

#### Jsonnet

**Import Syntax:**

```jsonnet
// Local file import
local lib = import 'library.libsonnet';

// Standard library (implicit)
local base64 = std.base64;

// String import (raw file content)
local schema = importstr 'schema.json';

// Binary import
local image = importbin 'logo.png';
```

**Resolution Algorithm:**

- Relative path resolution: `import 'file.libsonnet'` looks relative to importing file
- Library path search: `-J` flag adds directories to search path
- Standard library implicitly available as `std`
- No remote imports (file system only)
- Import paths resolved at compile time

**Version Pinning:**

- No built-in version management
- Version control via file system (git submodules, vendor directories)
- Some tools (e.g., jsonnet-bundler) add dependency management:

```jsonnet
// jsonnetfile.json
{
  "version": 1,
  "dependencies": [
    {
      "source": {
        "git": {
          "remote": "https://github.com/grafana/grafonnet-lib.git",
          "subdir": "grafonnet"
        }
      },
      "version": "v0.1.0"
    }
  ]
}
```

**Circular Dependency Detection:**

- Import cycles detected at parse/evaluation time
- Error reported when cycle is encountered
- All imports must form DAG

**Package Registry:**

- No official package registry
- Community convention: GitHub repositories
- Third-party tools (jsonnet-bundler) provide package management

**Strengths:**
- Simple import model
- Good for templating (powerful than pure JSON)
- importstr/importbin for non-Jsonnet content
- Mature, stable language

**Weaknesses:**
- No remote imports (security decision, but limits composability)
- No official package registry or version management
- Relies on external tools for dependency management
- Library paths can be ambiguous

**Sources:**
- [Jsonnet - Tutorial](https://jsonnet.org/learning/tutorial.html)
- [Jsonnet - Language Reference](https://jsonnet.org/ref/language.html)
- [Jsonnet - Specification](https://jsonnet.org/ref/spec.html)

---

### 1.5 Circular Dependency Detection Algorithms

**Depth-First Search (DFS):**

Most common approach for cycle detection:

```
visited = set()
recursion_stack = set()

function detectCycle(node):
    if node in recursion_stack:
        return true  // Cycle detected

    if node in visited:
        return false

    visited.add(node)
    recursion_stack.add(node)

    for dependency in node.dependencies:
        if detectCycle(dependency):
            return true

    recursion_stack.remove(node)
    return false
```

- **Time complexity**: O(V + E) where V = vertices (modules), E = edges (dependencies)
- **Space complexity**: O(V) for visited and recursion stack
- Detects back edges in directed graph

**Tarjan's Strongly Connected Components (SCC):**

Most efficient algorithm for finding all cycles:

- Finds all strongly connected components in O(V + E) time
- Linear time complexity in graph size
- Identifies which modules are part of cycles
- Can report all circular dependencies in a single pass

```
index = 0
stack = []
indices = {}
lowlinks = {}
on_stack = {}
sccs = []

function strongconnect(v):
    indices[v] = index
    lowlinks[v] = index
    index += 1
    stack.push(v)
    on_stack[v] = true

    for w in v.dependencies:
        if w not in indices:
            strongconnect(w)
            lowlinks[v] = min(lowlinks[v], lowlinks[w])
        elif on_stack[w]:
            lowlinks[v] = min(lowlinks[v], indices[w])

    if lowlinks[v] == indices[v]:
        scc = []
        while true:
            w = stack.pop()
            on_stack[w] = false
            scc.append(w)
            if w == v:
                break
        sccs.append(scc)
```

**Topological Sort (Kahn's Algorithm):**

Alternative approach that also detects cycles:

```
in_degree = {}  // Count of dependencies for each node
queue = []

// Initialize in-degree counts
for node in graph:
    in_degree[node] = count(node.dependencies)
    if in_degree[node] == 0:
        queue.add(node)

ordered = []
while queue not empty:
    node = queue.remove()
    ordered.append(node)

    for dependent in nodes_depending_on(node):
        in_degree[dependent] -= 1
        if in_degree[dependent] == 0:
            queue.add(dependent)

if len(ordered) != len(graph):
    // Cycle detected - some nodes still have dependencies
    return error("circular dependency")
```

- If topological sort can't order all nodes, a cycle exists
- Also O(V + E) time complexity
- Provides ordering for free if no cycles exist

**Practical Implementation Recommendations:**

1. **For simple cycle detection**: DFS with recursion stack (easiest to implement)
2. **For finding all cycles**: Tarjan's SCC algorithm (most efficient)
3. **For ordered resolution**: Kahn's topological sort (dual purpose)

**Sources:**
- [Circular dependency - Wikipedia](https://en.wikipedia.org/wiki/Circular_dependency)
- [Dependency Resolving Algorithm](https://www.electricmonk.nl/docs/dependency_resolving_algorithm/dependency_resolving_algorithm.html)
- [How to Handle Circular Dependencies](https://algocademy.com/blog/how-to-handle-circular-dependencies-a-comprehensive-guide/)
- [Replace cycle detection with Tarjan's SCC](https://github.com/aackerman/circular-dependency-plugin/pull/49)

---

### 1.6 Recommendations for IntentLang

Based on the research, here are recommendations for IntentLang's import system:

#### Import Syntax

```intentlang
// Local relative imports
import "./skills/search.ias"
import "../shared/types.ias"

// Versioned package imports (registry)
import "agentspec/web-tools" version "1.2.0"
import "github.com/user/agents" version "2.1.0"

// Aliased imports
import search from "./skills/search.ias"
import web from "agentspec/web-tools" version "1.2.0"
```

**Rationale:**
- Relative paths for local composition (like Terraform, Jsonnet)
- URI-style package names for registry imports (like Go, CUE)
- Explicit versioning (like Terraform, Go modules)
- Optional aliases for disambiguation

#### Resolution Algorithm

**Adopt Go's Minimal Version Selection (MVS) with modifications:**

1. **Local imports**: Resolve relative to importing file
2. **Package imports**: Resolve via registry with MVS
3. **Deterministic resolution**: Same inputs always produce same dependency tree
4. **No lock file needed**: MVS is deterministic by design

**Resolution order:**
1. Check if import is relative path (./ or ../)
   - If yes: resolve relative to current file
2. Check if import is package reference (has domain or registry prefix)
   - If yes: resolve via package registry using MVS
3. Check built-in packages (if any)

**Why MVS over other approaches:**
- Deterministic without lock files (unlike Terraform)
- Simple to understand and predict (unlike SAT solvers)
- Proven at scale (Go ecosystem)
- Minimal surprise (uses minimum required versions, not latest)

#### Version Pinning Mechanism

**Package manifest (.ias files):**
```intentlang
package "my-agent" version "1.0.0" lang "2.0"

// No explicit require - dependencies discovered from imports
```

**Dependency lock (.agentspec.lock):**
```json
{
  "version": "1.0",
  "packages": {
    "agentspec/web-tools": {
      "version": "1.2.0",
      "hash": "sha256:abc123...",
      "dependencies": {
        "agentspec/http-client": "0.5.0"
      }
    }
  }
}
```

**Version constraints:**
```intentlang
import "agentspec/web-tools" version "^1.2.0"  // >= 1.2.0, < 2.0.0
import "agentspec/core" version "~1.2.0"       // >= 1.2.0, < 1.3.0
import "agentspec/utils" version "1.2.0"       // Exact version
```

**Rationale:**
- Lock file for reproducibility (can be auto-generated)
- Semantic versioning constraints (familiar to developers)
- Cryptographic hashes for integrity (like Dhall, Go)

#### Circular Dependency Detection

**Use Tarjan's SCC algorithm during validation:**

1. Build dependency graph during parse phase
2. Run Tarjan's SCC after all imports resolved
3. Report all circular dependencies in a single pass
4. Provide clear error messages with import chain

```
Error: Circular dependency detected
  agent-a.ias -> shared.ias -> agent-b.ias -> agent-a.ias

Hint: Consider extracting shared definitions to a separate module
```

**Implementation in agentspec validate:**
- Track imports in AST node metadata
- Build directed graph of file dependencies
- Run SCC detection before semantic validation
- Exit early if cycles detected (cannot proceed with validation)

**Rationale:**
- O(V + E) efficiency (fast even for large projects)
- Detects all cycles in one pass
- Better error messages than simple DFS

#### Package Registry Pattern

**Adopt Go's GOPROXY protocol with IntentLang-specific endpoints:**

```
# Registry discovery
GET /.well-known/agentspec.json
  -> { "registry_url": "https://registry.agentspec.io/v1" }

# List versions
GET /v1/{package}/@v/list
  -> ["1.0.0", "1.1.0", "1.2.0"]

# Get version metadata
GET /v1/{package}/@v/{version}.info
  -> { "version": "1.2.0", "time": "2026-02-28T10:00:00Z" }

# Get package manifest
GET /v1/{package}/@v/{version}.spec
  -> { "package": "web-tools", "version": "1.2.0", ... }

# Download package
GET /v1/{package}/@v/{version}.zip
  -> (zip archive with .ias files)
```

**Local cache:**
- `~/.agentspec/pkg/` for downloaded packages
- `~/.agentspec/cache/` for metadata
- Cache keyed by package@version (immutable)

**Private registry support:**
```intentlang
// Registry configuration in .agentspec.toml
[registries]
default = "https://registry.agentspec.io"

[registries.private]
url = "https://registry.internal.company.com"
auth = "token"  // or "basic", "cert"
```

**Rationale:**
- HTTP-based (simple, cacheable, proxy-friendly)
- Registry protocol proven at scale (Go, npm)
- Supports private registries (enterprise requirement)
- Immutable packages (version = specific content)

#### Summary Table

| Feature | Recommendation | Inspiration |
|---------|---------------|-------------|
| **Import syntax** | Relative paths + URI packages | Go + Terraform |
| **Resolution** | MVS algorithm | Go modules |
| **Versioning** | Semantic versioning | Go + npm |
| **Lock file** | Optional .agentspec.lock | Go sum |
| **Cycle detection** | Tarjan's SCC | Computer Science |
| **Registry** | HTTP-based GOPROXY-like | Go proxy |
| **Security** | Cryptographic hashes | Go + Dhall |
| **Local imports** | Relative paths | Terraform + Jsonnet |

---

## Topic 2: Sandboxed Expression Evaluation in Go

### 2.1 google/cel-go (Common Expression Language)

**Overview:**

CEL is a non-Turing complete expression language designed for simplicity, speed, safety, and portability. Developed by Google and used across Google Cloud Platform services.

**Syntax Example:**

```javascript
// Property access
input.category == "support"

// Nested properties
input.metadata.type == "urgent"

// Comparisons
input.priority > 5 && input.status != "closed"

// Type checks
has(input.tags) && type(input.tags) == list

// Built-in functions
input.email.matches('[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}')
```

**Go Integration:**

```go
import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
)

// Create environment
env, _ := cel.NewEnv(
    cel.Declarations(
        decls.NewVar("input", decls.NewMapType(
            decls.String, decls.Dyn,
        )),
    ),
)

// Compile expression
ast, _ := env.Compile(`input.category == "support"`)

// Create evaluator
prg, _ := env.Program(ast)

// Evaluate
result, _ := prg.Eval(map[string]interface{}{
    "input": map[string]interface{}{
        "category": "support",
    },
})
```

**Syntax Readability:**

- **Excellent for non-programmers**: C-like syntax familiar from JavaScript/JSON
- Natural property access with dot notation
- Logical operators use words (and, or) or symbols (&&, ||)
- Clear error messages with position information
- Rich standard library (strings, math, lists, maps, timestamps)

**Performance:**

- Linear evaluation time: O(n) where n = expression size + input size
- Compiled to optimized AST representation
- Orders of magnitude faster than sandboxed JavaScript
- Benchmark: ~90 ns/op for simple comparisons (2026 benchmarks)

**Safety/Sandboxing:**

- **Non-Turing complete**: No loops, no recursion, guaranteed termination
- **No side effects**: Pure expression evaluation only
- **Type safe**: Static type checking at compile time
- **Controlled function access**: Only built-ins and explicitly provided functions
- **Resource limits**: Configurable evaluation cost limits
- **Safe for user code**: Designed to execute untrusted expressions

**Go Integration Quality:**

- Excellent: First-class Go library
- Strong typing integrates with Go types
- Protobuf support (native interop)
- Custom function extensions
- Good documentation and examples

**Maturity:**

- **Very mature**: Used in production at Google since 2018
- Active development and maintenance
- Large community and ecosystem
- Latest release: January 29, 2026
- Extensive test coverage
- Used in: Kubernetes (CEL for admission control), Google Cloud IAM, Envoy proxy

**Pros:**
- Battle-tested at Google scale
- Strong safety guarantees
- Excellent documentation
- Kubernetes integration (familiar to DevOps)
- Rich type system

**Cons:**
- More complex than needed for simple use cases
- Protobuf-centric design (can be overkill)
- Larger dependency footprint
- Learning curve for advanced features

**Sources:**
- [GitHub - google/cel-go](https://github.com/google/cel-go)
- [CEL-Go Codelab](https://codelabs.developers.google.com/codelabs/cel-go)
- [cel-go module](https://pkg.go.dev/github.com/google/cel-go)
- [Common Expression Language spec](https://github.com/google/cel-spec)

---

### 2.2 expr-lang/expr (previously antonmedv/expr)

**Overview:**

A Go-centric expression language designed for dynamic configurations with focus on performance, safety, and Go integration. Used by Google, Uber, ByteDance, and Alibaba.

**Syntax Example:**

```javascript
// Property access
input.category == "support"

// Comparisons and logic
input.priority > 5 && input.status != "closed"

// Built-in operators
input.metadata.type in ["urgent", "critical"]

// Ternary operator
input.priority >= 5 ? "high" : "low"

// Array/map operations
all(input.tags, # startsWith("prod-"))
```

**Go Integration:**

```go
import "github.com/expr-lang/expr"

// Compile expression
program, _ := expr.Compile(
    `input.category == "support"`,
    expr.Env(map[string]interface{}{
        "input": struct {
            Category string
            Priority int
        }{},
    }),
)

// Evaluate
output, _ := expr.Run(program, map[string]interface{}{
    "input": map[string]interface{}{
        "category": "support",
        "priority": 10,
    },
})
```

**Syntax Readability:**

- **Excellent for non-programmers**: JavaScript-like syntax
- Natural and intuitive operators
- Ternary operator for conditional expressions
- Array comprehensions (all, any, filter, map)
- String templates and interpolation
- Clear, concise error messages

**Performance:**

- **Fastest of all libraries**: ~70 ns/op for simple expressions
- 23% faster than CEL in benchmarks
- Optimizing compiler with bytecode VM
- Compile once, run many times (concurrent-safe)
- Minimal memory overhead
- Performance-critical applications at scale (Uber, Alibaba)

**Safety/Sandboxing:**

- **Memory safe**: No access to unrelated memory
- **Side-effect free**: Expressions only compute outputs from inputs
- **Always terminating**: No infinite loops guaranteed
- **Sandboxed execution**: Cannot cause side effects
- **Type safe**: Static type checking at compile time
- **Controlled environment**: Only exposed functions callable

**Go Integration Quality:**

- **Excellent**: Designed specifically for Go
- Seamless Go struct integration
- Native Go types support
- Easy custom function registration
- Idiomatic Go API
- Extensive examples and documentation

**Maturity:**

- **Mature**: Active development since 2018
- 7,000+ GitHub stars
- Used in production by major companies:
  - Google Cloud Platform
  - Uber (Uber Eats marketplace customization)
  - ByteDance
  - Alibaba (recommendation services)
- Regular releases and maintenance
- Comprehensive test suite
- Good community support

**Pros:**
- Best performance
- Go-native design
- Simple, intuitive syntax
- Large production deployments
- Minimal dependencies
- Easy to extend

**Cons:**
- Smaller ecosystem than CEL
- Less formal specification
- Fewer integrations with other tools

**Sources:**
- [GitHub - expr-lang/expr](https://github.com/expr-lang/expr)
- [Expression Language for Go](https://expr-lang.org/)
- [Expr Lang: Go centric expression language](https://wundergraph.com/blog/expr-lang-go-centric-expression-language)
- [Performance Comparison](https://github.com/antonmedv/golang-expression-evaluation-comparison)

---

### 2.3 Performance Comparison: CEL vs Expr

**Benchmark Results (2026):**

| Operation | expr | cel-go | Winner |
|-----------|------|--------|--------|
| Simple comparison | 70 ns/op | 91 ns/op | expr (23% faster) |
| String operations | 362 ns/op | 466 ns/op | expr (22% faster) |
| Complex expressions | ~500 ns/op | ~700 ns/op | expr (29% faster) |

**Real-world performance characteristics:**

**expr-lang/expr:**
- Optimizing compiler with bytecode VM
- Compile once, run many times pattern
- Minimal reflection overhead
- Direct Go type integration

**google/cel-go:**
- Linear evaluation O(n)
- Still very fast (orders faster than JS)
- More generic type system (protobuf-based)
- Slightly more overhead for type conversions

**Conclusion:**
- Both are fast enough for most use cases
- expr has edge for high-throughput scenarios
- CEL's overhead is negligible except at extreme scale

**Sources:**
- [golang-expression-evaluation-comparison](https://github.com/antonmedv/golang-expression-evaluation-comparison)
- [Benchmark Results govaluate vs cel-go](https://gist.github.com/rhnvrm/db4567fcd87b2cb8e997999e1366d406)

---

### 2.4 Custom Recursive Descent Evaluator

**Overview:**

Build a custom expression evaluator using recursive descent parsing. Full control over syntax, features, and behavior.

**Implementation Approach:**

```go
type Expr interface {
    Eval(ctx Context) (interface{}, error)
}

// Binary expression: left op right
type BinaryExpr struct {
    Left  Expr
    Op    string
    Right Expr
}

func (b *BinaryExpr) Eval(ctx Context) (interface{}, error) {
    left, err := b.Left.Eval(ctx)
    if err != nil {
        return nil, err
    }
    right, err := b.Right.Eval(ctx)
    if err != nil {
        return nil, err
    }

    switch b.Op {
    case "==":
        return left == right, nil
    case ">":
        return toNumber(left) > toNumber(right), nil
    case "and":
        return toBool(left) && toBool(right), nil
    // ...
    }
}

// Property access: object.property
type PropertyExpr struct {
    Object   Expr
    Property string
}

func (p *PropertyExpr) Eval(ctx Context) (interface{}, error) {
    obj, err := p.Object.Eval(ctx)
    if err != nil {
        return nil, err
    }
    return getProperty(obj, p.Property)
}
```

**Parsing Example:**

```go
type Parser struct {
    tokens []Token
    pos    int
}

func (p *Parser) parseExpression() Expr {
    return p.parseLogicalOr()
}

func (p *Parser) parseLogicalOr() Expr {
    left := p.parseLogicalAnd()

    for p.match("or") {
        op := p.previous()
        right := p.parseLogicalAnd()
        left = &BinaryExpr{left, op.Value, right}
    }

    return left
}

func (p *Parser) parseComparison() Expr {
    left := p.parsePrimary()

    if p.match("==", "!=", ">", "<", ">=", "<=") {
        op := p.previous()
        right := p.parsePrimary()
        return &BinaryExpr{left, op.Value, right}
    }

    return left
}
```

**Syntax Readability:**

- **Full control**: Define exactly what syntax you want
- Can make it as simple or complex as needed
- Can optimize for non-programmer audience
- Custom error messages tailored to use case

**Performance:**

- **Very fast**: Direct Go code, minimal overhead
- Can be optimized for specific use cases
- No generic type system overhead
- Potential to be fastest option if well-implemented

**Safety/Sandboxing:**

- **Requires careful implementation**:
  - Must manually prevent infinite loops (no while/for constructs)
  - Must control which functions are callable
  - Must validate property access
  - Must handle recursion limits
  - Must implement type safety

**Go Integration:**

- **Native**: It's all Go code
- Can integrate exactly as needed
- Full control over type conversions
- Direct access to Go types

**Maturity:**

- **N/A**: Would be new implementation
- No existing ecosystem
- Full testing burden
- Documentation burden
- Maintenance burden

**Examples:**

- [svstanev/goexp](https://github.com/svstanev/goexp) - Recursive descent expression parser
- [bediger4000/arithmetic-parser](https://github.com/bediger4000/arithmetic-parser) - Arithmetic expression parser
- [Recursive Descent Parsing guide](http://gokcehan.github.io/notes/recursive-descent-parsing.html)

**Pros:**
- Complete control over features
- Can be optimized for exact use case
- No external dependencies
- Lightweight (only what you need)
- Learning opportunity

**Cons:**
- Significant development time
- Testing burden (edge cases, security)
- Maintenance burden
- No community or ecosystem
- Security risks if not carefully implemented
- Potential for bugs

**Sources:**
- [GitHub - svstanev/goexp](https://github.com/svstanev/goexp)
- [Recursive descent parser - expression calculator](https://dev.to/arxeiss/recursive-descent-parser-5581)
- [A Practical guide to building a parser in Go](https://gagor.pro/2026/01/a-practical-guide-to-building-a-parser-in-go/)
- [Parsing Expressions by Recursive Descent](https://www.engr.mun.ca/~theo/Misc/exp_parsing.htm)

---

### 2.5 Recommendations for IntentLang

**Primary Recommendation: expr-lang/expr**

After evaluating all options, **expr-lang/expr** is the best choice for IntentLang's runtime control flow expressions.

#### Rationale:

**1. Performance:**
- Fastest option (70 ns/op vs CEL's 91 ns/op)
- Critical for high-throughput agent runtimes
- Proven at scale (Uber, Alibaba)

**2. Syntax Readability:**
- JavaScript-like syntax familiar to most developers
- Intuitive for non-programmers
- Clear, natural operators
- Good error messages

```javascript
// Very readable for non-programmers
input.category == "support" && input.priority >= 5

// Ternary for simple conditionals
input.urgent ? "high" : "normal"

// Array operations
any(input.tags, # startsWith("prod-"))
```

**3. Safety:**
- Sandboxed execution (no side effects)
- Always terminating (no infinite loops)
- Memory safe
- Type safe with static checking

**4. Go Integration:**
- Designed specifically for Go
- Seamless Go struct integration
- Easy custom function registration
- Idiomatic API

```go
// Easy to integrate
program, _ := expr.Compile(
    `input.category == "support"`,
    expr.Env(Input{}),
)
result, _ := expr.Run(program, env)
```

**5. Maturity:**
- Production-proven at major companies
- Active maintenance
- Good documentation
- Large community (7k+ stars)

**6. Minimal Dependencies:**
- Small footprint
- No heavy dependencies
- Easy to vendor

#### Alternative Consideration: google/cel-go

**Use CEL if:**
- Kubernetes integration is important (CEL is used in K8s admission control)
- Need strong protobuf integration
- Want Google's backing and formal specification
- Prefer slightly more conservative choice

**Trade-offs:**
- Slightly slower (but still very fast)
- More complex API
- Larger dependency footprint

#### Not Recommended: Custom Recursive Descent

**Reasons:**
- Significant development time
- Security burden (must get sandboxing right)
- Testing and maintenance burden
- No community support
- Risk of bugs and vulnerabilities

**Only consider if:**
- Expr and CEL both have show-stopping limitations
- Need extremely specialized syntax
- Have resources for proper implementation and security review

---

### 2.6 Expression Use Cases in IntentLang

**Pipeline Step Conditionals:**

```intentlang
pipeline "data-processing" {
  step "validate" {
    agent "validator"
    input "raw_data"
    when "input.size > 0 && input.format == 'json'"
  }

  step "transform" {
    agent "transformer"
    depends_on ["validate"]
    when "steps.validate.status == 'success'"
  }
}
```

**Agent Delegation:**

```intentlang
agent "router" {
  delegate to agent "support" when "input.category == 'support'"
  delegate to agent "sales" when "input.category == 'sales'"
  delegate to agent "technical" when "input.priority >= 8"
}
```

**Conditional Tool Execution:**

```intentlang
skill "conditional-search" {
  description "Search with conditional execution"
  input {
    query string required
    use_cache bool
  }

  execution {
    when "input.use_cache && cache.has(input.query)"
      return cache.get(input.query)

    when "!input.use_cache || !cache.has(input.query)"
      call tool "search" with input.query
  }
}
```

**Required Expression Features:**

1. **Property access**: `input.category`, `input.metadata.type`
2. **Comparisons**: `==`, `!=`, `>`, `<`, `>=`, `<=`
3. **Boolean logic**: `&&`, `||`, `!` (and, or, not)
4. **Type checks**: `type(x) == string`, `has(input.field)`
5. **String operations**: `startsWith()`, `contains()`, `matches()`
6. **Array operations**: `in`, `all()`, `any()`
7. **Ternary operator**: `condition ? true_val : false_val`

**All of these are supported by expr-lang/expr out of the box.**

---

### 2.7 Integration Example

**Example: Using expr in AgentSpec Runtime**

```go
package runtime

import (
    "github.com/expr-lang/expr"
    "github.com/expr-lang/expr/vm"
)

// ConditionEvaluator evaluates when clauses in pipeline steps
type ConditionEvaluator struct {
    programs map[string]*vm.Program
}

// Compile pre-compiles all condition expressions
func (e *ConditionEvaluator) Compile(conditions map[string]string) error {
    e.programs = make(map[string]*vm.Program)

    for name, condition := range conditions {
        program, err := expr.Compile(
            condition,
            expr.Env(map[string]interface{}{
                "input": map[string]interface{}{},
                "steps": map[string]interface{}{},
            }),
            expr.AsBool(), // Ensure boolean result
        )
        if err != nil {
            return fmt.Errorf("invalid condition %q: %w", name, err)
        }
        e.programs[name] = program
    }

    return nil
}

// Evaluate executes a pre-compiled condition
func (e *ConditionEvaluator) Evaluate(name string, ctx Context) (bool, error) {
    program, ok := e.programs[name]
    if !ok {
        return false, fmt.Errorf("condition %q not found", name)
    }

    env := map[string]interface{}{
        "input": ctx.Input,
        "steps": ctx.Steps,
    }

    result, err := expr.Run(program, env)
    if err != nil {
        return false, err
    }

    return result.(bool), nil
}

// Usage in pipeline runtime:
func (r *PipelineRuntime) ExecuteStep(step *PipelineStep, ctx Context) error {
    if step.When != "" {
        shouldRun, err := r.evaluator.Evaluate(step.Name, ctx)
        if err != nil {
            return fmt.Errorf("condition evaluation failed: %w", err)
        }
        if !shouldRun {
            log.Printf("Skipping step %s (condition not met)", step.Name)
            return nil
        }
    }

    // Execute step...
    return r.executeAgent(step.Agent, ctx)
}
```

**Benefits of this approach:**
- Compile expressions once during initialization
- Fast evaluation at runtime (70 ns/op)
- Type-safe environment configuration
- Clear error messages for invalid conditions
- No security risks (sandboxed execution)

---

## Summary Comparison Table

### Expression Evaluators

| Feature | expr-lang/expr | google/cel-go | Custom Parser |
|---------|----------------|---------------|---------------|
| **Performance** | 70 ns/op ⭐ | 91 ns/op | Variable |
| **Syntax** | JavaScript-like ⭐ | C-like | Custom |
| **Sandboxing** | Excellent ⭐ | Excellent ⭐ | Requires work |
| **Go Integration** | Excellent ⭐ | Very good | Native |
| **Maturity** | Production-proven ⭐ | Production-proven ⭐ | N/A |
| **Dependencies** | Minimal ⭐ | Moderate | None ⭐ |
| **Type Safety** | Yes ⭐ | Yes ⭐ | Manual |
| **Ecosystem** | Good | Excellent | None |
| **Learning Curve** | Low ⭐ | Medium | N/A |

**Recommendation: expr-lang/expr** (best balance of performance, safety, and ease of use)

---

## Appendix: All Sources

### Topic 1: Import/Module Systems

**Go Modules:**
- [Minimal Version Selection](https://research.swtch.com/vgo-mvs)
- [The Principles of Versioning in Go](https://research.swtch.com/vgo-principles)
- [Go Modules Reference](https://go.dev/ref/mod)
- [Go Module Proxy](https://proxy.golang.org/)
- [Go proxy for GitLab](https://docs.gitlab.com/user/packages/go_proxy/)
- [Mastering Go Modules](https://dev.to/leapcell/mastering-go-modules-a-practical-guide-to-dependency-management-3ccb)

**Terraform Modules:**
- [Use modules in your configuration](https://developer.hashicorp.com/terraform/language/modules/configuration)
- [Module Sources](https://developer.hashicorp.com/terraform/language/modules/sources)
- [Dependency Lock File](https://developer.hashicorp.com/terraform/language/files/dependency-lock)
- [Terraform Lock Files Explained](https://spacelift.io/blog/terraform-lock-hcl-file)

**Protobuf:**
- [Language Guide (proto 3)](https://protobuf.dev/programming-guides/proto3/)
- [Manage dependencies - Buf Docs](https://buf.build/docs/bsr/module/dependency-management/)
- [Protocol Buffers Language Specification](https://protobuf.dev/reference/protobuf/proto3-spec/)

**Dhall:**
- [Safety Guarantees](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)
- [The Dhall configuration language](https://dhall-lang.org/)
- [Safety guarantees GitHub](https://github.com/dhall-lang/dhall-lang/blob/master/docs/discussions/Safety-guarantees.md)

**CUE:**
- [Modules, Packages, and Instances](https://cuelang.org/docs/concept/modules-packages-instances/)
- [Imports](https://cuelang.org/docs/tour/packages/imports/)
- [Proposal: package management](https://github.com/cue-lang/cue/issues/851)
- [CUE Modules](https://cuelang.org/docs/reference/modules/)

**Jsonnet:**
- [Jsonnet Tutorial](https://jsonnet.org/learning/tutorial.html)
- [Jsonnet Language Reference](https://jsonnet.org/ref/language.html)
- [Jsonnet Specification](https://jsonnet.org/ref/spec.html)

**Comparisons:**
- [Comparisons between CUE, Jsonnet, Dhall, OPA](https://github.com/cue-lang/cue/discussions/669)
- [Taming the Beast: Comparing Jsonnet, Dhall, Cue](https://pv.wtf/posts/taming-the-beast)

**Circular Dependencies:**
- [Circular dependency - Wikipedia](https://en.wikipedia.org/wiki/Circular_dependency)
- [Dependency Resolving Algorithm](https://www.electricmonk.nl/docs/dependency_resolving_algorithm/dependency_resolving_algorithm.html)
- [How to Handle Circular Dependencies](https://algocademy.com/blog/how-to-handle-circular-dependencies-a-comprehensive-guide/)
- [Tarjan's SCC Algorithm PR](https://github.com/aackerman/circular-dependency-plugin/pull/49)

### Topic 2: Expression Evaluation

**google/cel-go:**
- [GitHub - google/cel-go](https://github.com/google/cel-go)
- [CEL-Go Codelab](https://codelabs.developers.google.com/codelabs/cel-go)
- [cel-go module](https://pkg.go.dev/github.com/google/cel-go)
- [Common Expression Language spec](https://github.com/google/cel-spec)

**expr-lang/expr:**
- [GitHub - expr-lang/expr](https://github.com/expr-lang/expr)
- [Expression Language for Go](https://expr-lang.org/)
- [Expr Lang: Go centric expression language](https://wundergraph.com/blog/expr-lang-go-centric-expression-language)
- [Performance Comparison](https://github.com/antonmedv/golang-expression-evaluation-comparison)

**Custom Parsers:**
- [GitHub - svstanev/goexp](https://github.com/svstanev/goexp)
- [Recursive descent parser - calculator](https://dev.to/arxeiss/recursive-descent-parser-5581)
- [Practical guide to building a parser in Go](https://gagor.pro/2026/01/a-practical-guide-to-building-a-parser-in-go/)
- [Parsing Expressions by Recursive Descent](https://www.engr.mun.ca/~theo/Misc/exp_parsing.htm)

---

**End of Research Document**
