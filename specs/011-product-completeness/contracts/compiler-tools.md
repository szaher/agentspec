# Compiler Tool Generation Contract

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## Tool Types

Each compiler target MUST generate functional tool implementations for these tool types:

### HTTP Tools

**Input**: `tool http { url "<url>" method "<METHOD>" }`

**Generated code must**:
1. Import appropriate HTTP client library for the target framework
2. Make an HTTP request to the configured URL with the configured method
3. Return the response body as a string
4. Include basic error handling (connection failure, non-2xx status)

### Command Tools

**Input**: `tool command { binary "<binary>" args "<args>" }`

**Generated code must**:
1. Import subprocess/exec library for the target language
2. Execute the configured binary with arguments
3. Capture and return stdout
4. Include error handling (binary not found, non-zero exit)

### Inline Tools

**Input**: `tool inline { language "<lang>" code "<code>" }`

**Generated code must**:
1. Embed the inline code as a function body or subprocess call
2. For Python targets: inline the code directly if language is Python
3. For non-matching languages: call the appropriate interpreter as a subprocess
4. Include a comment noting the source language

### Unsupported/Missing Tool Config

**Generated code must**:
1. Include a clear `# TODO:` comment explaining what needs implementation
2. Raise an appropriate error (not silently return empty string)
3. Reference the original skill name for context

## Target-Specific Patterns

### CrewAI (Python)
- HTTP: `urllib.request.Request` + `urlopen`
- Command: `subprocess.run([binary, *args], capture_output=True, text=True)`
- Inline: Direct Python code or `subprocess.run([interpreter, "-c", code])`

### LangGraph (Python)
- HTTP: `requests.get/post` or `urllib.request`
- Command: `subprocess.run`
- Inline: Same as CrewAI

### LlamaIndex (Python)
- HTTP: `requests` or `urllib.request`
- Command: `subprocess.run`
- Inline: Same as CrewAI

### LlamaStack (Python)
- HTTP: `urllib.request` (minimal dependencies)
- Command: `subprocess.run`
- Inline: Same as CrewAI

## Test Contract

Each target MUST pass these test scenarios:
1. Compile agent with HTTP tool → generated code contains HTTP client call, not "not implemented"
2. Compile agent with command tool → generated code contains subprocess call, not "not implemented"
3. Compile agent with inline tool → generated code contains inline execution, not "not implemented"
4. Compile agent with unknown tool type → generated code contains TODO comment and raises error
