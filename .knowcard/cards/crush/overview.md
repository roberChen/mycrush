---
id: d7258c359558845d253838bfa337fc3a
title: Crush Project Overview
keywords:
    - crush
    - overview
    - architecture
    - dependencies
    - go
    - charm
    - fantasy
    - bubbletea
    - terminal AI assistant
summary: Crush is a terminal-based AI coding assistant built in Go by Charm. It connects to LLMs and gives them tools to read, write, and execute code. Supports multiple providers, LSPs, MCP servers, hooks, and agent skills.
created: 2026-07-10T19:23:26.943308358Z
updated: 2026-07-10T19:23:26.943308358Z
---

## Overview

Crush is a terminal-based AI coding assistant by [Charm](https://charm.land). It connects LLMs to tools for reading, writing, and executing code.

**Module path:** `github.com/charmbracelet/crush`
**Go version:** 1.26.4
**Build flags:** `CGO_ENABLED=0`, `GOEXPERIMENT=greenteagc`

## Key Dependencies

| Dependency | Role |
|---|---|
| `charm.land/fantasy` | LLM provider abstraction (Anthropic, OpenAI, Gemini, Bedrock, etc.) |
| `charm.land/bubbletea/v2` | TUI framework (Bubble Tea v2, Elm architecture) |
| `charm.land/lipgloss/v2` | Terminal styling |
| `charm.land/glamour/v2` | Markdown rendering |
| `charm.land/catwalk` | Snapshot/golden-file testing for TUI |
| `charm.land/fang/v2` | CLI framework (cobra-based) |
| `sqlc` | SQL → Go code generation for database layer |
| `mvdan.cc/sh/v3` | Pure-Go POSIX shell interpreter |
| `github.com/modelcontextprotocol/go-sdk` | MCP client integration |
| `pressly/goose/v3` | Database migrations |
| `ncruces/go-sqlite3` / `modernc.org/sqlite` | SQLite drivers (WASM + pure Go) |

## Supported Providers

Anthropic, OpenAI, Google Gemini, AWS Bedrock, GitHub Copilot, Hyper (Charm's own), Vercel AI Gateway, OpenRouter, Azure, Vertex AI, Z.ai, and any OpenAI- or Anthropic-compatible API.

## Key Features

- Multi-model support, switch LLMs mid-session
- Session-based conversations with SQLite persistence
- LSP integration for code intelligence
- MCP extensibility (stdio, http, sse)
- Agent skills system (builtin + user-defined)
- Hooks system (PreToolUse shell commands)
- Custom agentic tools
- Client/server mode for remote operation
- Works on macOS, Linux, Windows (PowerShell/WSL), Android, FreeBSD
