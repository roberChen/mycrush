---
id: c82dd3b2a689567b288d2ed6d4b8df77
title: 'LSP Integration: Manager, Auto-Discovery, and Diagnostics'
keywords:
    - LSP
    - language server protocol
    - gopls
    - manager
    - auto-discovery
    - diagnostics
    - references
    - powernap
    - on-demand
summary: The LSP system uses a lazy Manager with three discovery layers (powernap defaults, user config, runtime file-type matching). On-demand startup with 30s timeout. Versioned diagnostics caching with settle-wait pattern. Integrates with edit/view tools for real-time feedback.
created: 2026-07-10T19:24:45.395050622Z
updated: 2026-07-10T19:24:45.395050622Z
---

## LSP Manager (`internal/lsp/manager.go`)

The `Manager` struct manages LSP client lifecycle:

```go
type Manager struct {
    clients     *csync.Map[string, *Client]
    unavailable *csync.Map[string, time.Time]  // failed servers + backoff
    cfg         *config.ConfigStore
    manager     *powernapconfig.Manager        // powernap server registry
    callback    func(name string, client *Client)
}
```

### Three Discovery Layers

1. **Built-in defaults (powernap)** — `LoadDefaults()` loads catalog of known servers (gopls, rust-analyzer, pyright, etc.) with commands, file types, root markers
2. **User configuration** — Merged from `cfg.Config().LSP` (override or disable)
3. **Runtime file-type matching** — `handles(server, filePath, workDir)` checks extension + root markers

### On-Demand Startup (`Start(ctx, path)`)

`startServer` ordering (cheap checks first):
1. Auto-LSP check (skip if not user-configured and auto-LSP disabled)
2. Already-running check
3. Unavailability backoff (30s retry delay for failed servers)
4. Binary existence check (`exec.LookPath`)
5. Generic command blocklist (python, node, npx, java, etc.)
6. File-type/root-marker match (expensive, done last)
7. Client creation + initialization (30s timeout)
8. Poll `IsRunning()` every 500ms until ready

All matching servers start in parallel via `sync.WaitGroup`.

### Auto-LSP Gating
- `Options.AutoLSP` = `nil` or `true` → auto-start allowed
- `false` → only explicitly configured LSPs start
- `skipAutoStartCommands` blocklist prevents ambiguous commands from auto-starting

## Diagnostics

### Storage
Each `Client` maintains `csync.VersionedMap[protocol.DocumentURI, []protocol.Diagnostic]` — versioned concurrent map for efficient caching.

### Ingestion
`textDocument/publishDiagnostics` handler updates the map, invokes `onDiagnosticsChanged` callback.

### Cached Counts
`GetDiagnosticCounts()` uses version to avoid recomputing severity counts on every UI render.

### Settle-Wait Pattern
`WaitForDiagnostics(ctx, timeout)`:
1. Wait up to 1s for first diagnostics change
2. Then wait 300ms of stability (no new changes)

## Tool Integration

| Tool | LSP Usage |
|---|---|
| `edit` / `multiedit` / `write` | `notifyLSPs()` — opens file, notifies change, waits for diagnostics |
| `view` | `openInLSPs()` + `waitForLSPDiagnostics()` — read-only |
| `lsp_diagnostics` | Aggregates from all clients |
| `lsp_references` | `FindReferences()` via matching client |
| `lsp_restart` | `client.Restart()` |

LSP tools only registered when LSPs are configured or auto-LSP enabled.

## Workspace Edit Handling
`workspace/applyEdit` handler lets LSP servers apply code actions (auto-fixes, refactoring) directly to filesystem. Supports UTF-8/UTF-16/UTF-32 position encodings.
