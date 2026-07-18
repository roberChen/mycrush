---
id: 918eb096e2593b7e40ce27df666624db
title: Workspace, Server, and Client Architecture
keywords:
    - workspace
    - server
    - client
    - backend
    - client-server
    - SSE
    - HTTP API
    - remote mode
    - AppWorkspace
    - ClientWorkspace
summary: Workspace interface abstracts local vs remote operation. AppWorkspace delegates to in-process app.App. ClientWorkspace proxies via HTTP+SSE. Server exposes REST API. Backend manages workspace lifecycle and SSE stream refcounting. Supports Unix sockets, named pipes, and TCP.
created: 2026-07-10T19:26:50.272916067Z
updated: 2026-07-10T19:26:50.272916067Z
---

## Workspace Abstraction (`internal/workspace/`)

The `Workspace` interface is the central API consumed by all frontends (TUI/CLI). Abstracts whether running in-process or against a remote server.

```go
type Workspace interface {
    // ~60 methods covering: sessions, messages, agent, permissions,
    // filetracker, history, LSP, MCP, config, skills, events
}
```

### Two Implementations

| Implementation | Mode | Description |
|---|---|---|
| `AppWorkspace` | Local/in-process | Delegates directly to `app.App` services |
| `ClientWorkspace` | Remote client/server | Proxies via HTTP client SDK + SSE events |

### Event Subscription (ClientWorkspace)
`Subscribe(program)` opens SSE event stream → `consumeEvents()` reads in loop → `translateEvent()` maps proto types to domain types → `program.Send()` injects into Bubble Tea.

## Server Mode (`internal/server/`)

HTTP server exposing RESTful API (`/v1/...`) with full CRUD for workspaces, sessions, messages, agents, permissions, config, LSP, filetracker.

- **Transports:** Unix sockets (Linux/macOS), named pipes (Windows), TCP
- **SSE:** Event streaming to connected clients
- **Graceful shutdown:** Managed lifecycle

Key types: `Server`, `NewServer`, `DefaultServer`, `Start()`, `Shutdown()`

## Backend (`internal/backend/`)

Transport-agnostic business logic between HTTP server and `app.App`.

- **Workspace management:** Create/delete by path dedup, tracks connected clients
- **SSE stream refcounting:** Per-client stream count, hold timer
- **Agent dispatch:** Coordinates agent runs across clients
- **Shutdown:** Triggers when last workspace removed

```go
type Backend struct {
    workspaces map[string]*Workspace  // path → workspace
    pathIndex  map[string]string      // dedup
    config     *config.ConfigStore
    shutdown   func()
}
```

## Client (`internal/client/`)

HTTP client SDK for talking to a Crush server.

- **Dial:** Unix sockets, named pipes, TCP
- **Typed methods:** Health, config, workspaces, sessions, messages, agent runs, SSE events, file uploads

```go
type Client struct {
    http.Client
    network string  // "unix", "tcp", "npipe"
    address string
    clientID string
}
```

## How It All Fits Together

```
Local Mode:
  CLI → AppWorkspace → app.App → services (agent, db, lsp, mcp, ...)

Server Mode:
  CLI → ClientWorkspace → HTTP → Server → Backend → app.App → services
                                         ↓
                                    SSE events → Client → program.Send()
```

The `crush run` command starts in client mode, connecting to a background server process.
