---
id: 3a2131e508a32703e8691de6cb9e1094
title: 'Hooks System: PreToolUse Shell Commands'
keywords:
    - hooks
    - PreToolUse
    - hook runner
    - decision
    - allow
    - deny
    - halt
    - Claude Code
    - exit codes
    - shell commands
summary: 'The hook system runs user-defined shell commands before tool calls (PreToolUse). Returns decisions: allow, deny, or halt. Supports input rewriting and context injection. Runs hooks in parallel via embedded POSIX shell. Compatible with both Crush native and Claude Code output formats.'
created: 2026-07-10T19:24:02.822119507Z
updated: 2026-07-10T19:24:02.822119507Z
---

## Overview

The hook system lets users define shell commands that fire before tool calls (`PreToolUse`), returning decisions that control agent behavior. Package: `internal/hooks/`.

## Key Types (`hooks.go`)

```go
type Decision int  // DecisionNone, DecisionAllow, DecisionDeny

type HookResult struct {
    Decision     Decision
    Halt         bool    // halt the entire turn
    Reason       string  // deny/halt reason
    Context      string  // extra context to inject
    UpdatedInput string  // JSON patch against tool_input
}
```

## Execution (`runner.go`)

`Runner.Run(ctx, event, sessionID, toolName, toolInputJSON) → AggregateResult`

1. **Match** — Filters hooks by regex `Matcher` against tool name
2. **Dedup** — Identical `Command` strings deduplicated (first wins)
3. **Parallel execution** — All hooks run as goroutines via `sync.WaitGroup`
4. **Aggregate** — Results merged: Deny > Allow > None; Halt is sticky

### Exit Code Handling
| Exit Code | Result |
|-----------|--------|
| 0 | Parse stdout JSON for decision |
| 2 | `DecisionDeny` + stderr as reason |
| 49 | `DecisionDeny` + `Halt=true` + stderr as reason |
| Other non-zero | Non-blocking warning, `DecisionNone` |
| Timeout | `DecisionNone` (goroutine abandoned after 1s grace) |

Default timeout: **30 seconds** per hook.

## I/O Format (`input.go`)

### Stdin Payload (JSON)
```json
{
  "event": "PreToolUse",
  "session_id": "...",
  "cwd": "...",
  "tool_name": "bash",
  "tool_input": { ... }
}
```

### Environment Variables
`CRUSH_EVENT`, `CRUSH_TOOL_NAME`, `CRUSH_SESSION_ID`, `CRUSH_CWD`, `CRUSH_PROJECT_DIR`, plus `CRUSH_TOOL_INPUT_COMMAND` and `CRUSH_TOOL_INPUT_FILE_PATH` extracted from tool input.

### Output Formats
**Crush native:**
```json
{"version": 1, "decision": "allow|deny", "halt": false, "reason": "...", "context": "...", "updated_input": {...}}
```

**Claude Code compatible:**
```json
{"hookSpecificOutput": {"permissionDecision": "allow|deny", "permissionDecisionReason": "...", "updatedInput": {...}}}
```

## Integration

- `hookedTool` decorator wraps tools at coordinator level (top-level only, **never in sub-agents**)
- Hook `Allow` → pre-approves permission via `permission.WithHookApproval(ctx)`
- Hook `Deny` → blocks tool call with error
- Hook `Halt` → blocks AND ends entire turn (`StopTurn = true`)
- Hook `UpdatedInput` → rewrites tool input before execution
- Hook `Context` → appends extra context to tool result

## Config
```json
{
  "hooks": {
    "PreToolUse": [
      {"name": "lint", "matcher": "edit|write", "command": "golangci-lint run", "timeout": 60}
    ]
  }
}
```

Hooks run through Crush's embedded POSIX shell (mvdan/sh) — same builtins/coreutils as bash tool.
