---
id: 38c9ee5d8d8df13b1babbc71b05016ad
title: Pub/Sub Messaging and Message Model
keywords:
    - pubsub
    - broker
    - events
    - message model
    - content parts
    - debounce
    - streaming
    - event types
summary: 'In-process pub/sub broker with two delivery modes: lossy Publish for streaming deltas, bounded-blocking PublishMustDeliver for terminal events. Services embed Broker directly. Message model has 8 content part types. Message Service uses 33ms debounce with terminal-event bypass for reliable delivery.'
created: 2026-07-10T19:26:12.849045815Z
updated: 2026-07-10T19:26:12.849045815Z
---

## Pub/Sub Broker (`internal/pubsub/broker.go`)

Generic, in-process fan-out broker. Each subscriber gets a buffered channel (size 4096).

### Two Delivery Semantics

| Method | Semantics | Use Case |
|--------|-----------|----------|
| `Publish(t, payload)` | **Best-effort, lossy** — non-blocking, drops if full | High-frequency streaming deltas (token-by-token) |
| `PublishMustDeliver(ctx, t, payload)` | **Bounded-blocking** — tries non-blocking, then blocks up to 50ms per subscriber | Terminal events (finish, error, cancel) |

Drops counted via `DropCount()` / `MustDeliverDropCount()`.

### Lifecycle
- `Subscribe(ctx)` — Creates channel, spawns goroutine to unsubscribe on ctx done
- `Shutdown()` — Closes all channels

### Event Types
```go
CreatedEvent = "created"
UpdatedEvent = "updated"
DeletedEvent = "deleted"
```
Payload type constants: `message`, `session`, `file`, `agent_event`, `permission_request`, `lsp_event`, `mcp_event`, `config_changed`, `skills_event`, `run_complete`

### Usage Pattern
Broker embedded directly into services as `*pubsub.Broker[T]`:

| Service | Broker Type |
|---------|------------|
| Message Service | `Broker[Message]` |
| Session Service | `Broker[Session]` |
| File History | `Broker[File]` |
| Permission Service | `Broker[PermissionRequest]` |
| App Events | `Broker[tea.Msg]` |
| LSP Events | `Broker[LSPEvent]` |
| MCP Tools | `Broker[Event]` |
| Skills Manager | `Broker[Event]` |

## Message Model (`internal/message/`)

### Content Part Types (all implement `ContentPart` interface)

| Type | Purpose |
|------|---------|
| `TextContent` | Plain text |
| `ReasoningContent` | Chain-of-thought (provider-specific signatures) |
| `ImageURLContent` | Image by URL |
| `BinaryContent` | Raw binary (base64 for OpenAI) |
| `ToolCall` | Tool invocation (ID, name, input, finished) |
| `ToolResult` | Tool call result (content, data, MIME, error) |
| `Finish` | Terminal marker with FinishReason |
| `ShellCommand` | Bang-mode shell command + output |

### Serialization
Parts serialized via `partWrapper{Type, Data}` discrimination. JSON tags: `"text"`, `"reasoning"`, `"tool_call"`, etc.

### AI Provider Conversion
`ToAIMessage()` converts internal `Message` → `[]fantasy.Message` for LLM API calls.

### Message Service (`message.go`)
Embeds `*pubsub.Broker[Message]`:
- **Debounced updates** — 33ms debounce window coalesces streaming token deltas
- **Terminal flush** — bypasses debounce for structural changes (finish, tool call, reasoning end), uses `PublishMustDeliver`
- **Flush control** — `Flush(id)`, `FlushAll()` for shutdown/session-switch ordering
