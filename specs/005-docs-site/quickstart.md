# Quickstart: AgentSpec Documentation Site

**Branch**: `005-docs-site` | **Date**: 2026-02-23

## Prerequisites

- Python 3.9+ (for MkDocs)
- Go 1.25+ (for building `agentspec` and running validation tests)
- Git

## Local Development

### 1. Install MkDocs and dependencies

```bash
pip install mkdocs-material pymdown-extensions

# Install the custom IntentLang Pygments lexer
pip install -e docs-tools/pygments_intentlang/
```

### 2. Serve locally with hot reload

```bash
mkdocs serve
```

Visit `http://localhost:8000`. Pages auto-reload on save.

### 3. Validate code examples

```bash
# Build the agentspec binary
go build -o agentspec ./cmd/agentspec

# Run documentation example validation
go test ./integration_tests/ -run TestDocExamples -v
```

### 4. Build the static site

```bash
mkdocs build --strict
```

Output is in `site/` directory.

## Writing Documentation

### Adding a new page

1. Create a Markdown file in the appropriate `docs/` subdirectory
2. Add the file to the `nav` section in `mkdocs.yml`
3. Use the fence tag conventions for code examples:
   - ` ```ias ` for complete, valid examples
   - ` ```ias fragment ` for fragments
   - ` ```ias novalidate ` for pseudocode

### Adding a use case

1. Create a `.ias` example file in `docs/examples/`
2. Validate it: `./agentspec validate docs/examples/my-example.ias`
3. Create a use case page in `docs/user-guide/use-cases/`
4. Include the example with a Mermaid architecture diagram

### Adding a CLI command page

1. Create a page in `docs/user-guide/cli/`
2. Document usage syntax, all flags, examples, and output
3. Extract flag information from `cmd/agentspec/<command>.go`

## Deployment

### Manual deploy

```bash
mkdocs gh-deploy
```

### CI deploy (automatic)

Push to `main` branch. The GitHub Actions workflow:
1. Builds `agentspec` binary
2. Validates all `.ias` examples in docs
3. Builds the site
4. Deploys to GitHub Pages

## Testing Scenarios

### Scenario 1: New user onboarding
- Follow `docs/user-guide/getting-started/quickstart.md` end-to-end
- Verify all code examples are copy-pasteable and valid
- Verify "next steps" links work

### Scenario 2: Language reference lookup
- Navigate to any language reference page (e.g., `agent`)
- Verify syntax definition, attribute table, and examples are present
- Copy an example to a `.ias` file and run `agentspec validate`

### Scenario 3: Use case evaluation
- Browse `docs/user-guide/use-cases/index.md` comparison table
- Click into a use case (e.g., ReAct)
- Verify diagram, complete example, and deployment instructions

### Scenario 4: Deployment guide
- Navigate to a deployment target (e.g., Kubernetes)
- Verify prerequisites, `deploy` block example, and health check config

### Scenario 5: Search
- Search for "pipeline" — verify results include pipeline language reference and pipeline use case
- Search for "execution command" (deprecated) — verify migration guidance appears

### Scenario 6: Site build validation
- Run `mkdocs build --strict`
- Verify zero warnings (no broken links, no missing pages)
- Run `go test ./integration_tests/ -run TestDocExamples`
- Verify all code examples pass validation
