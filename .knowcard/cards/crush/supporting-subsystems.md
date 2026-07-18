---
id: ccaa6d78f5ecd8d9ae6922921462e16b
title: 'Supporting Subsystems: Session, History, Telemetry, OAuth'
keywords:
    - session
    - history
    - filetracker
    - event
    - telemetry
    - clipboard
    - oauth
    - update
    - herdr
    - PostHog
    - supporting systems
summary: 'Supporting subsystems: Session CRUD, versioned file history, file read tracking, PostHog telemetry, cross-platform clipboard, OAuth2 for Copilot/Hyper, GitHub release update checker, and herdr terminal multiplexer integration.'
created: 2026-07-10T19:26:50.534923826Z
updated: 2026-07-10T19:26:50.534923826Z
---

## Supporting Subsystems

### Session Service (`internal/session/`)
Session CRUD backed by SQLite. Manages chat session lifecycle, token usage/cost tracking, per-session todos. Supports parent/child sessions for agent tool calls.
- Key types: `Session`, `Service` (Create, Get, List, Save, Delete, Rename)
- `Todo` / `TodoStatus` for task tracking within sessions

### History Service (`internal/history/`)
Versioned file content storage. Stores snapshots of file contents the agent reads/writes with auto-incrementing version numbers.
- Key types: `File`, `Service` (Create, CreateVersion, Get, ListBySession)

### File Tracker (`internal/filetracker/`)
Lightweight tracking of when files were read within a session. Stores timestamp per (session, file path) pair. Paths relative to working directory.
- Key types: `Service` (RecordRead, LastReadTime, ListReadFiles)

### Event/Telemetry (`internal/event/`)
Anonymous usage analytics via PostHog (`data.charm.land`). Sends session/prompt/token/error events. Includes machine identifier management.
- Key functions: `Init()`, `Flush()`, `SessionCreated()`, `PromptSent()`, `TokensUsed()`

### Clipboard (`internal/clipboard/`)
Cross-platform clipboard with build-tag guards. Real read/write on supported platforms, no-op stubs on Android/iOS. Supports text and image (PNG).
- Key functions: `Init()`, `WriteText()`, `Read(Format)`

### OAuth (`internal/oauth/`)
OAuth2 token management. Provider sub-packages:
- **copilot/** — GitHub Copilot device flow + disk persistence + custom HTTP transport
- **hyper/** — Hyper device flow auth
- Key type: `Token` (AccessToken, RefreshToken, ExpiresAt, IsExpired())

### Update Checker (`internal/update/`)
Checks GitHub Releases API for new versions. Compares current vs latest with pre-release detection.
- Key function: `Check(ctx, current, client) → Info`

### herdr Integration (`internal/herdr/`)
Integration with herdr terminal multiplexer. Reports agent state (idle/working/blocked) over Unix socket. Auto-detects via `HERDR_ENV`, `HERDR_SOCKET_PATH`, `HERDR_PANE_ID` env vars.
- Key type: `Client`, `HandleEvent(Event)`, `Translate(ev) → Event`
- Nil when not in herdr environment
