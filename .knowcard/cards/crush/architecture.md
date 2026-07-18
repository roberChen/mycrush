---
id: 3db32bf4e5ce1c7aabba890b9603cdaa
title: Crush Architecture and Codebase Layout
keywords:
    - architecture
    - directory structure
    - packages
    - patterns
    - internal
    - app wiring
    - codebase layout
summary: High-level architecture of Crush showing the directory structure, key packages, and design patterns. The app.go file wires together all services (DB, config, agents, LSP, MCP, events, permissions).
created: 2026-07-10T19:23:27.822957674Z
updated: 2026-07-10T19:23:27.822957674Z
---

## Directory Structure

```
main.go                            CLI entry point
internal/
  app/app.go                       Top-level wiring: DB, config, agents, LSP, MCP, events
  cmd/                             CLI commands (root, run, login, models, stats, sessions)
  config/                          Config struct, loading, provider configuration
  agent/                           SessionAgent + Coordinator (LLM conversation orchestration)
    coordinator.go                 Manages named agents ("coder", "task")
    agent.go                       SessionAgent: runs LLM conversations per session
    hooked_tool.go                 Decorator: runs PreToolUse hooks before tool execution
    prompts.go                     Loads Go-template system prompts
    templates/                     System prompt templates (*.md.tpl)
    tools/                         All built-in tools (bash, edit, view, grep, glob, etc.)
      mcp/                         MCP client integration
    customtools/                   Custom agentic tool definitions (TOOL.md)
  hooks/                           Hook engine: PreToolUse shell commands
  session/session.go               Session CRUD (SQLite)
  message/                         Message model + content types + service
  db/                              SQLite via sqlc + goose migrations
  lsp/                             LSP client manager, auto-discovery
  ui/                              Bubble Tea v2 TUI
  permission/                      Tool permission checking + allow-lists
  skills/                          Skill file discovery and loading
  shell/                           Bash execution via mvdan/sh + background jobs
  pubsub/                          In-process pub/sub broker
  workspace/                       Workspace abstraction (local + remote)
  server/                          HTTP server mode (REST API + SSE)
  backend/                         Transport-agnostic business logic
  client/                          HTTP client SDK for server mode
  event/                           Telemetry (PostHog)
  filetracker/                     Tracks files read per session
  history/                         Versioned file snapshots
  clipboard/                       Cross-platform clipboard
  oauth/                           OAuth2 (Copilot, Hyper)
  herdr/                           herdr terminal multiplexer integration
```

## Key Architectural Patterns

1. **Config is a Service** — accessed via `config.Service`/`config.ConfigStore`, not global state. Copy-on-write with atomic swaps.
2. **Tools are self-documenting** — each tool has `.go` + `.md` in `internal/agent/tools/`
3. **System prompts are Go templates** — `internal/agent/templates/*.md.tpl`
4. **Context files** — reads AGENTS.md, CRUSH.md, CLAUDE.md, GEMINI.md from working dir
5. **SQLite + sqlc** — all queries in `internal/db/sql/`, generated code in `internal/db/`
6. **Pub/sub** — `internal/pubsub` for decoupled agent ↔ UI ↔ services messaging
7. **Hooks** — PreToolUse shell commands, independent engine, wraps tools at coordinator level
8. **CGO disabled** — builds with `CGO_ENABLED=0`
