# Contract: Session Store (Redis)

**Feature**: 008-state-data-integrity
**Scope**: `internal/session/redis_store.go` — SaveMessages/LoadMessages

## Interface Changes

### SaveMessages(ctx context.Context, sessionID string, messages []llm.Message) error

**Current behavior** (broken):
```
GET session:{id}:messages → unmarshal → append → marshal → SET session:{id}:messages
```
Race condition: two concurrent saves can lose messages.

**New behavior**:
```
RPUSH session:{id}:messages <msg1_json> <msg2_json> ...
EXPIRE session:{id}:messages <ttl>
```

**Guarantees**:
- Atomic append (Redis RPUSH is atomic)
- O(1) per message regardless of history size
- TTL refreshed on each save
- Error returned to caller on failure (never silently discarded)

### LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error)

**Current behavior** (broken):
```
GET session:{id}:messages → unmarshal entire JSON array
Returns nil on error (silently discards)
```

**New behavior**:
```
LRANGE session:{id}:messages 0 -1 → unmarshal each element individually
```

**Guarantees**:
- Returns all messages in insertion order
- Returns error on Redis failure (not nil)
- Individual message unmarshal failures logged at WARN, skipped (partial results returned)

## Migration

Existing sessions may use string-based storage (single JSON value under `GET` key).

**Detection**: On `LoadMessages`, if key type is `string` (not `list`):
1. Unmarshal the string value as `[]llm.Message`
2. Delete the string key
3. RPUSH each message to the new list key
4. Log INFO "migrated session messages to list format"

This migration is best-effort and happens transparently on first access.

## Redis Key Schema

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `session:{id}` | hash | Session metadata (existing, unchanged) |
| `session:{id}:messages` | list | Message list (changed from string to list) |

## Error Handling

- Redis connection errors: return wrapped error to caller
- Individual message marshal/unmarshal errors: log WARN, skip element
- RPUSH failure: return error, existing messages remain intact (Redis guarantees)
