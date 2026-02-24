# Parser Pipeline

The parser converts IntentLang `.ias` source files into a typed Abstract Syntax Tree (AST). It is implemented as a hand-written recursive-descent parser with a separate lexer stage.

## Packages

| Package | Path | Purpose |
|---------|------|---------|
| `parser` | `internal/parser/` | Lexer, token types, and recursive-descent parser |
| `ast` | `internal/ast/` | AST node type definitions |

## Pipeline Stages

```text
  .ias source string
        |
        v
  +------------+
  |   Lexer    |  Tokenize() -> []Token
  +------+-----+
         |
         v
  +------------+
  |   Parser   |  Parse() -> *ast.File
  +------+-----+
         |
         v
    AST (ast.File)
```

### 1. Lexer (Tokenization)

The lexer (`internal/parser/lexer.go`) reads the raw source string and produces a flat slice of tokens. Each token carries its type, literal value, source file path, line, and column.

Key token categories:

- **Keywords** -- `package`, `agent`, `prompt`, `skill`, `secret`, `deploy`, `server`, `client`, `pipeline`, `policy`, `plugin`, `type`, `import`, `binding`, `environment`, and field-level keywords like `model`, `uses`, `content`, `strategy`, etc.
- **Delimiters** -- `{`, `}`, `[`, `]`, `(`, `)`, `,`
- **Literals** -- strings (double-quoted), numbers (integer and float), booleans (`true`, `false`)
- **Identifiers** -- unquoted names used in metadata keys and value positions
- **Newlines** -- significant for statement separation
- **EOF** -- marks end of input

Token types are defined in `internal/parser/token.go`:

```go
type TokenType string

const (
    TokenEOF     TokenType = "EOF"
    TokenNewline TokenType = "NEWLINE"
    TokenString  TokenType = "STRING"
    TokenNumber  TokenType = "NUMBER"
    TokenIdent   TokenType = "IDENT"
    TokenLBrace  TokenType = "{"
    TokenRBrace  TokenType = "}"
    // ... keyword tokens
    TokenPackage TokenType = "package"
    TokenAgent   TokenType = "agent"
    TokenPrompt  TokenType = "prompt"
    // etc.
)
```

### 2. Parser (Recursive Descent)

The parser (`internal/parser/parser.go`) consumes the token stream and builds the AST. Entry point:

```go
func Parse(input, file string) (*ast.File, []*ParseError)
```

This function:

1. Creates a `Lexer` and calls `Tokenize()` to get the token slice.
2. Constructs a `Parser` with the token slice and an empty name registry.
3. Calls `parseFile()` which dispatches to resource-specific parse methods.
4. Returns the AST and any accumulated parse errors.

The parser's top-level dispatch (`parseStatement()`) switches on the current token to determine which resource to parse:

```go
func (p *Parser) parseStatement() ast.Statement {
    switch {
    case p.check(TokenAgent):   return p.parseAgent()
    case p.check(TokenPrompt):  return p.parsePrompt()
    case p.check(TokenSkill):   return p.parseSkill()
    case p.check(TokenDeploy):  return p.parseDeployTarget()
    case p.check(TokenSecret):  return p.parseSecret()
    case p.check(TokenServer):  return p.parseMCPServer()
    case p.check(TokenClient):  return p.parseMCPClient()
    case p.check(TokenPolicy):  return p.parsePolicy()
    case p.check(TokenPlugin):  return p.parsePlugin()
    case p.check(TokenPipeline): return p.parsePipeline()
    case p.check(TokenTypeKw):  return p.parseTypeDef()
    // ...
    }
}
```

### 3. AST Construction

The AST (`internal/ast/ast.go`) is a tree of typed nodes rooted at `ast.File`. Every node implements the `Node` interface:

```go
type Node interface {
    Pos() Pos  // Start position in source
    End() Pos  // End position in source
}
```

Top-level statements implement the `Statement` interface (which embeds `Node`):

```go
type Statement interface {
    Node
    stmtNode()  // marker method
}
```

The main AST node types:

| Node | Description |
|------|-------------|
| `File` | Root node containing `Package` header and `[]Statement` |
| `Package` | Package declaration with name, version, lang version |
| `Agent` | Agent resource with model, prompt ref, skill refs, runtime config |
| `Prompt` | Prompt resource with content and template variables |
| `Skill` | Skill resource with I/O schema and tool configuration |
| `MCPServer` | MCP server definition (stdio or HTTP transport) |
| `MCPClient` | MCP client connecting to servers |
| `Secret` | Secret reference (env or store source) |
| `Policy` | Policy with allow/deny/require rules |
| `DeployTarget` | Deployment configuration (target platform, resources, health) |
| `Pipeline` | Multi-step agent workflow with dependency graph |
| `TypeDef` | Named type definition (struct, enum, or list) |
| `Binding` | Legacy adapter binding (IntentLang 1.0) |
| `Environment` | Environment overlay with attribute overrides |

## Source Positions

Every AST node carries `StartPos` and `EndPos` fields of type `ast.Pos`:

```go
type Pos struct {
    File   string  // Source file path
    Line   int     // 1-based line number
    Column int     // 1-based column number
}
```

Positions are used for:

- **Error messages** -- The parser formats errors as `file:line:col: error: message`.
- **Formatter** -- The canonical formatter uses positions to preserve or normalize whitespace.
- **IDE integration** -- Source maps for diagnostics and go-to-definition.

## Error Recovery

Parse errors are accumulated rather than causing immediate failure. The parser uses two mechanisms:

1. **Error collection** -- `addError(msg, hint)` appends a `ParseError` to the parser's error list and continues parsing. The hint field provides actionable guidance.

2. **Token skipping** -- When the parser encounters an unexpected token inside a block, it advances past it and continues. This allows reporting multiple errors in a single parse pass.

```go
type ParseError struct {
    File    string
    Line    int
    Column  int
    Message string
    Hint    string  // Optional actionable suggestion
}

func (e *ParseError) Error() string {
    s := fmt.Sprintf("%s:%d:%d: error: %s", e.File, e.Line, e.Column, e.Message)
    if e.Hint != "" {
        s += "\n  hint: " + e.Hint
    }
    return s
}
```

## Duplicate Detection

The parser maintains a name registry (`map[string]map[string]bool`) keyed by resource kind. When a resource name is registered, the parser checks for duplicates within the same kind:

```go
func (p *Parser) registerName(kind, name string, pos ast.Pos) {
    if p.names[kind][name] {
        p.addError(
            fmt.Sprintf("duplicate %s name %q", kind, name),
            "each resource must have a unique name within its kind",
        )
    }
    p.names[kind][name] = true
}
```

This catches errors like defining two agents with the same name before the validation stage.

## Adding a New Resource Type

To add a new top-level resource type:

1. Define the AST node in `internal/ast/ast.go` implementing `Statement`.
2. Add a token for the keyword in `internal/parser/token.go`.
3. Add the keyword to the lexer's keyword map in `internal/parser/lexer.go`.
4. Add a `case` in `parseStatement()` that calls your new parse method.
5. Write the parse method following the existing pattern: consume keyword, read name, parse brace-delimited body.
6. Add validation rules in `internal/validate/`.
7. Add IR lowering logic in `internal/ir/`.
