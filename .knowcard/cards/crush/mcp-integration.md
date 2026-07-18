---
id: 1943a18f13d4bce336296d5127a0182f
title: 'MCP Integration: Client Lifecycle and Tool Bridging'
keywords:
    - MCP
    - Model Context Protocol
    - stdio
    - http
    - sse
    - tools
    - resources
    - prompts
    - transport
    - client session
summary: MCP integration manages client lifecycle with package-level global state. Supports stdio/http/sse transports with 15s timeout and auto-renew. Tools discovered via ListTools, filtered by allow/deny lists. MCP tools bridged to fantasy.AgentTool with namespaced names (mcp_{server}_{tool}). Resources and prompts also accessible.
created: 2026-07-10T19:26:13.123375718Z
updated: 2026-07-10T19:26:13.123375718Z
---

## MCP Integration (`internal/agent/tools/mcp/`)

Uses **package-level global state** (concurrent-safe maps + pubsub broker) rather than structs.

### Initialization (`init.go`)

`Initialize(ctx, permissions, cfg)` — called once at app startup:
1. Iterates `cfg.Config().MCP` map
2. Skips `Disabled` entries
3. Spawns goroutine per enabled MCP → `initClient()`
4. `WaitForInit()` blocks until all complete

### Transport Types
| Type | Transport |
|------|-----------|
| `stdio` | `mcp.CommandTransport` (subprocess) |
| `http` | `mcp.StreamableClientTransport` |
| `sse` | `mcp.SSEClientTransport` |

All config values pass through `VariableResolver` for `$VAR`/`$(cmd)` expansion.

### Client Lifecycle
```
StateDisabled → StateStarting → StateConnected
                            → StateError
```
- Default timeout: **15 seconds**
- Auto-renew: `getOrRenewClient()` pings before use, recreates session if ping fails
- `Close(ctx)` — closes all sessions concurrently

### Tool Discovery & Execution (`tools.go`)
- `getTools()` always calls `ListTools()` (spec compliance)
- `filterTools()` applies `EnabledTools` (allow-list) then `DisabledTools` (deny-list)
- `RefreshTools()` re-fetches on `EventToolsListChanged`

`RunTool(ctx, cfg, name, toolName, input)`:
- Parses input JSON → `map[string]any`
- Calls `session.CallTool()`
- Handles TextContent, ImageContent, AudioContent
- Returns `ToolResult{Type, Content, Data, MediaType}`

### Resources (`resources.go`)
- `ListResources()` — checks `Capabilities.Resources != nil` first
- `ReadResource()` — reads by URI
- Gracefully handles `Method not found` errors

### Prompts (`prompts.go`)
- `GetPromptMessages()` — extracts user-role TextContent messages only

### Events
```go
EventStateChanged         // connection state transitions
EventToolsListChanged     // server pushed tool list change
EventPromptsListChanged   // server pushed prompt list change
EventResourcesListChanged // server pushed resource list change
```

## MCP → fantasy.AgentTool Bridge (`mcp-tools.go`)

`GetMCPTools(permissions, cfg, wd) []*Tool` wraps each MCP tool:

- **Name:** `"mcp_{serverName}_{toolName}"` (namespaced)
- **Info:** Converts `InputSchema` → `fantasy.ToolInfo`
- **Run:** Permission check → `mcp.RunTool()` → map result to `fantasy.ToolResponse`
  - Docker MCP tools whitelisted (skip permission)
  - Image results only if model supports images

### Wiring
`coordinator.getToolsForAgent()` calls `GetMCPTools()`, applies per-agent `AllowedMCP` filtering.

Resource access tools also exposed:
- `list_mcp_resources` — `NewListMCPResourcesTool()`
- `read_mcp_resource` — `NewReadMCPResourceTool()`
