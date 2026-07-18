---
id: 144092d85742d95ca9d800d848e18ce5
title: 'App Wiring: Service Initialization and Lifecycle'
keywords:
    - app.go
    - wiring
    - initialization
    - App struct
    - services
    - event brokers
    - LSP callback
    - cleanup
    - InitCoderAgent
summary: 'app.New() wires all services: sessions, messages, history, permissions, filetracker, agent coordinator, LSP manager, skills manager, event brokers. Initialization is sequenced with background goroutines for MCP, updates, and LSP tracking. Cleanup functions ensure graceful shutdown.'
created: 2026-07-10T19:26:50.790792728Z
updated: 2026-07-10T19:26:50.790792728Z
---

## App Wiring (`internal/app/app.go`)

`app.New()` is the central wiring function that connects all services.

### Services Initialized

```go
type App struct {
    Sessions    session.Service
    Messages    message.Service
    History     history.Service
    Permissions permission.Service
    FileTracker filetracker.Service
    AgentCoordinator agent.Coordinator
    LSPManager  *lsp.Manager
    Skills      *skills.Manager
    config      *config.ConfigStore
    events      *pubsub.Broker[tea.Msg]
    agentNotifications *pubsub.Broker[notify.Notification]
    runCompletions     *pubsub.Broker[notify.RunComplete]
    herdrClient        *herdr.Client
}
```

### Initialization Sequence (`New()`)
1. Create SQLite queries (`db.New(conn)`)
2. Initialize services (sessions, messages, history, permissions, filetracker)
3. Create LSP manager
4. Set up event brokers
5. Initialize clipboard (best-effort)
6. Check for updates (background goroutine)
7. Initialize MCP clients (background goroutine: `mcp.Initialize`)
8. Start herdr integration
9. Register cleanup functions (DB release, MCP close)
10. Initialize coder agent (`InitCoderAgent`)
11. Set up LSP state callback
12. Start LSP config tracking (`TrackConfigured`)

### Event System
`app.events` is a `*pubsub.Broker[tea.Msg]` bridging all services to the UI:
- `Events(ctx)` — Returns subscription channel for UI
- `SendEvent(msg)` — Publish to all subscribers

### LSP State Management
Callback set on LSP manager:
- Client nil → `updateLSPState(name, StateUnstarted, ...)`
- Client ready → Set diagnostics callback, update state

### Agent Initialization (`InitCoderAgent`)
Creates the coordinator with all dependencies. Async initialization:
- System prompt building from template
- Tool set building via `buildTools()`

### Cleanup
Cleanup functions stored in `cleanupFuncs` slice, executed on shutdown:
- `db.Release(dataDir)` — Release shared DB connection
- `mcp.Close(ctx)` — Close all MCP sessions
