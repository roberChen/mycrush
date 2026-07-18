---
id: e58d2f54c75de9dd8c18e2f2654a9034
title: 'Shell System: mvdan/sh Execution and Background Jobs'
keywords:
    - shell
    - bash
    - mvdan/sh
    - POSIX
    - background jobs
    - process group
    - block functions
    - jq
    - coreutils
summary: 'Shell execution uses mvdan/sh (pure-Go POSIX interpreter). Two surfaces: stateful Shell (persists cwd/env) and stateless Run functions. Process group isolation prevents TTY hijacking. Background job manager (max 50, 8h retention). Handler chain: jq builtin → script dispatch → block list → coreutils.'
created: 2026-07-10T19:25:32.92283469Z
updated: 2026-07-10T19:25:32.92283469Z
---

## Shell System (`internal/shell/`)

Uses **mvdan.cc/sh/v3** — a pure-Go POSIX shell interpreter. Cross-platform, no `/bin/sh` required.

### Two Execution Surfaces

**Stateful (`Shell` struct):**
- `Exec(cmd) → (stdout, stderr, error)` — buffered output
- `ExecStream(cmd, stdout, stderr) → error` — live output writers
- Persists cwd and exported env vars across calls (`updateShellFromRunner`)
- Mutex-guarded

**Stateless (`Run` function):**
- `Run(ctx, RunOptions)` — parse + execute, no state retention
- `RunAndCapture` → `CaptureResult{Output, ExitCode}`
- `RunAndCapturePTY` → forces ANSI color env vars (`COLORTERM`, `CLICOLOR_FORCE`)
- `RunAndCaptureStream` → streaming via callback
- `RunAndPersist` → PTY path + PersistFunc callback

### Process Group Isolation (Unix)
- `processGroupExecHandler` sets `SysProcAttr.Setsid = true` (new session)
- Negative-PID signal targeting kills entire child process group
- Prevents zsh/bash job control from hijacking the terminal
- Windows: no-op, delegates to `DefaultExecHandler`

### Background Jobs (`background.go`)
- `BackgroundShellManager` — singleton via `sync.Once`
- Max 50 concurrent background jobs
- Hex IDs (`%03X` format)
- `syncBuffer` for thread-safe stdout/stderr capture
- `GetOutput(id)` — non-blocking read
- `Wait(id)` / `WaitContext(id, ctx)` — block until completion
- `Kill(id)` — cancel context, block on done
- `Cleanup()` — removes jobs older than 8 hours

### Handler Chain (ordered)
1. **`builtinHandler`** — in-process `jq` (gojq) wins over PATH binary
2. **`scriptDispatchHandler`** — path-prefixed argv[0]: shebang→exec, binary→passthrough, text→in-process shell
3. **`blockHandler`** — deny-list check (security)
4. **`coreutils.ExecHandler`** — Go-native coreutils (Windows-only by default)

### Block Functions
- `CommandsBlocker(cmds []string)` — blocks exact `argv[0]` matches
- `ArgumentsBlocker(cmd, args, flags)` — blocks specific command+args+flags combination
- Blocked commands return: `"command is not allowed for security reasons"`

### Environment
- `CRUSH=1`, `AGENT=crush`, `AI_AGENT=crush` markers injected
- Non-interactive env: `TERM=xterm-256color`, `EDITOR=false`, `PAGER=cat`
- `HERDR_*` vars stripped from child processes
- `ExpandValue()` expands `$VAR`, `${VAR}`, `$(cmd)` for config values
