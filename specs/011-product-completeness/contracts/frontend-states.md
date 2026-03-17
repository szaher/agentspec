# Frontend State Contract

**Feature**: 011-product-completeness
**Date**: 2026-03-17

## UI States

The built-in web frontend (`internal/frontend/web/`) MUST handle these states:

### 1. Loading State

**Trigger**: `fetchAgents()` called on page load or retry
**Display**: Centered loading spinner or pulse animation with "Connecting to AgentSpec..." text
**Duration**: Visible until fetch resolves or rejects
**Transition**: → Ready (on success) or → Error (on failure)

### 2. Error State

**Trigger**: `fetchAgents()` fails (network error, 4xx/5xx)
**Display**: Error banner at top of chat area with:
- Error icon
- Message: "Unable to connect to AgentSpec server" (or specific error)
- "Retry" button
**Behavior**: Retry button calls `fetchAgents()` again (transitions to Loading)
**Persistence**: Banner stays visible until retry succeeds or page reload

### 3. Empty State (Welcome)

**Trigger**: No messages in current session AND agents loaded successfully
**Display**: Centered welcome card with:
- AgentSpec logo or name
- "Welcome to AgentSpec"
- Brief instructions: "Select an agent from the dropdown and type a message to get started."
- Optional: list of example prompts as clickable chips
**Transition**: Disappears when first message is sent

### 4. Ready State (Existing)

**Trigger**: Agents loaded, messages may or may not exist
**Display**: Normal chat interface with message history, input area, agent selector

## Markdown Rendering

Already implemented via `renderMarkdown()` function. Must support:
- Headings (h1-h4)
- Code blocks (fenced with language hint)
- Inline code
- Bold, italic, strikethrough
- Lists (ordered and unordered)
- Links
- Tables
- Blockquotes
- Horizontal rules

## Test Scenarios

1. Open frontend with server running → loading indicator visible briefly → agents load → ready state
2. Open frontend with server down → loading indicator → error banner with retry button
3. Click retry with server now up → loading indicator → agents load → ready state
4. New session with no messages → welcome message with instructions visible
5. Send first message → welcome message disappears, chat begins
6. Receive markdown response → formatting preserved in chat bubble
