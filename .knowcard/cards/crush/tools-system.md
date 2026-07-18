---
id: fe85077b5e5400f9926d47ab6ee72c23
title: Built-in Tools Registry
keywords:
    - tools
    - bash
    - edit
    - view
    - grep
    - glob
    - ls
    - fetch
    - sourcegraph
    - LSP
    - MCP
    - todos
    - sub-agent
    - custom tools
summary: 'All built-in tools in Crush''s tool registry. Categories: core editing (bash, edit, multiedit, write, view, ls, glob, grep), web/network, job management, LSP, MCP, meta/debug, sub-agent, and custom agentic tools.'
created: 2026-07-10T19:24:02.522923819Z
updated: 2026-07-10T19:24:02.522923819Z
---

## Tool Categories

All tools live in `internal/agent/tools/`. Each tool has a `.go` implementation and a `.md` description file.

### Core Editing Tools
| Tool | Description |
|---|---|
| `bash` | Execute shell commands (background jobs, working dir, auto-timeout) |
| `edit` | Single find/replace in a file (exact match required) |
| `multiedit` | Multiple sequential find/replace on one file |
| `write` | Create or overwrite entire file |
| `view` | Read file with line numbers, offset/limit. Renders images. |
| `ls` | List directory tree with depth limit + ignore patterns |
| `glob` | Find files by glob pattern (sorted by mtime, max 100) |
| `grep` | Search file contents (regex/literal). Uses ripgrep when available. |

### Web/Network Tools
| Tool | Description |
|---|---|
| `fetch` | Fetch URL → markdown (max 100KB). Requires permission. |
| `web_fetch` | Simple web page fetcher for sub-agents (no permission) |
| `web_search` | Web search via DuckDuckGo for sub-agents |
| `download` | Download URL to local file (max 600s timeout) |
| `sourcegraph` | Search public GitHub repos via Sourcegraph API |

### Job Management
| Tool | Description |
|---|---|
| `job_output` | Get output from background shell job |
| `job_kill` | Terminate background shell job by ID |

### LSP Tools
| Tool | Description |
|---|---|
| `lsp_diagnostics` | Get LSP diagnostics for file or project |
| `lsp_references` | Find symbol references via LSP |
| `lsp_restart` | Restart one or all LSP clients |

### MCP Tools
| Tool | Description |
|---|---|
| `list_mcp_resources` | List resources from an MCP server |
| `read_mcp_resource` | Read resource by URI from MCP server |
| `mcp_{server}_{tool}` | Dynamic tools from MCP servers (namespaced) |

### Meta/Debug
| Tool | Description |
|---|---|
| `todos` | Session-level todo list management |
| `crush_info` | Return Crush environment info |
| `crush_logs` | Read recent Crush log entries |

### Sub-Agent Tools (in parent package)
| Tool | Description |
|---|---|
| `agent` | Spawn sub-agent with restricted tools (glob, grep, ls, view) |
| `agentic_fetch` | AI-powered web content analysis sub-agent |

### Custom Agentic Tools
User-defined tools from `TOOL.md` files. Each spawns a fresh sub-agent with custom system prompt, scoped skills, and configurable allowed tools/MCPs.
