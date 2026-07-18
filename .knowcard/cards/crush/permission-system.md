---
id: 546aaf2535d0a842af353dc19fa75052
title: 'Permission System: Tool Access Control'
keywords:
    - permission
    - allow-list
    - hook approval
    - interactive prompt
    - auto-approve
    - GrantPersistent
    - PermissionRequest
    - security
summary: 'Permission system uses a 6-stage cascade: skip → allow-list → hook approval → auto-approve session → persistent grant → interactive prompt. Race-safe via atomic channel Take. Persistent grants keyed by (session, tool, action, directory). Two pubsub channels for requests + notifications.'
created: 2026-07-10T19:25:33.183015105Z
updated: 2026-07-10T19:25:33.183015105Z
---

## Permission System (`internal/permission/`)

Single `Service` interface, one `permissionService` implementation.

### 6-Stage Short-Circuit Cascade (`Request()`)

| Stage | Check | Effect |
|-------|-------|--------|
| 1 | `skip.Load()` | Global skip mode → grant |
| 2 | **Allow-list** | `tool:action` or bare `tool` match → grant |
| 3 | **Hook approval** | PreToolUse hook said "allow" → grant + notify |
| 4 | **Auto-approve session** | Entire session auto-approved → grant |
| 5 | **Persistent grant** | `(session, tool, action, path)` previously granted → grant |
| 6 | **Interactive prompt** | Publish `PermissionRequest` via pubsub, block on response channel |

### Allow-List Matching
From config `permissions.allowed_tools`:
- `"bash"` — grants ALL actions for that tool
- `"bash:execute"` — grants only that specific action

### Hook Approval Integration
1. `hookedTool.Run()` runs PreToolUse hooks
2. If hook returns `Allow` → stamps context: `permission.WithHookApproval(ctx, call.ID)`
3. Tool calls `permissions.Request(ctx, ...)`
4. Stage 3 detects approval (scoped to exact tool call ID)

### Interactive Permission Flow
```
Tool → Request() → stages 1-5 miss → Stage 6:
  1. requestMu.Lock() — serialize one at a time
  2. Create PermissionRequest with UUID
  3. Register respCh in pendingRequests
  4. Publish PermissionRequest event → UI subscribes
  5. BLOCK on <-respCh
  
UI calls: Grant() / GrantPersistent() / Deny()
  → resolve(): atomic Take from pendingRequests
  → Publish PermissionNotification
  → Send on respCh (unblocks tool)
```

### Race Safety
`pendingRequests.Take(id)` atomically removes channel. First caller wins:
- Exactly one notification per resolution
- `GrantPersistent` only records if it wins the race
- Stray Grant/Deny for unknown IDs are safe no-ops

### Resolution Entry Points
1. **TUI/local** — `AppWorkspace` wraps `Grant`/`GrantPersistent`/`Deny`
2. **Backend/client-server** — maps proto actions to service methods
3. **Auto-approve** — `AutoApproveSession()` from app, custom tools, agentic_fetch

### Persistent Grant Key
```go
type PermissionKey struct {
    SessionID, ToolName, Action, Path string
}
```
`Path` resolved to **directory** — grant for `/tmp` covers all files in that dir.

### Two Pub/Sub Channels
- **Request broker** — UI/clients subscribe for prompts needing decisions
- **Notification broker** — broadcasts resolution outcomes for audit/UI
