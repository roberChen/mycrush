---
id: fdcc18c49ea7ccf8f52d501820cf5f09
title: 'Database Layer: SQLite, sqlc, and Schema'
keywords:
    - database
    - SQLite
    - sqlc
    - goose
    - migrations
    - schema
    - sessions
    - messages
    - files
    - persistence
summary: 'Crush uses SQLite with sqlc for type-safe queries and goose for migrations. 4 tables: sessions, messages, files (versioned snapshots), read_files. Single connection serialized, WAL mode, auto-applied embedded migrations.'
created: 2026-07-10T19:24:45.101387222Z
updated: 2026-07-10T19:24:45.101387222Z
---

## SQLite + sqlc + goose

Crush uses SQLite as its sole persistence layer. The DB file is `crush.db` in the configurable data directory.

### Connection Layer (`internal/db/`)
- **Two backends:** `modernc.org/sqlite` (pure Go, mainstream platforms) + `ncruces/go-sqlite3` (WASM, fallback)
- **Pragmas:** WAL mode, FOREIGN_KEYS=ON, BUSY_TIMEOUT=30000ms, 8MB cache
- **Pooling:** Global `pool map[string]*connEntry`, reference-counted. `SetMaxOpenConns(1)` serializes access.
- **Locking:** Optional `flock`-based exclusive lock on data dir (`crush.lock`)

### sqlc Configuration (`sqlc.yaml`)
- Schema: `internal/db/migrations/` (goose files double as schema)
- Queries: `internal/db/sql/*.sql` (5 files, annotated)
- Output: `internal/db/` package `db`
- Flags: `emit_json_tags`, `emit_prepared_queries`, `emit_interface`

## Schema: 4 Tables

### `sessions`
| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | UUID |
| `parent_session_id` | TEXT (nullable) | Self-ref FK for sub-agent sessions |
| `title` | TEXT | |
| `message_count` | INTEGER | Auto-maintained by triggers |
| `prompt_tokens` / `completion_tokens` | INTEGER | Cumulative usage |
| `cost` | REAL | |
| `summary_message_id` | TEXT (nullable) | Points to summary message |
| `todos` | TEXT (nullable) | JSON-serialized todo list |
| `inherited_message_count` | INTEGER | Messages from parent session |

### `messages`
| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | |
| `session_id` | TEXT FK→sessions | CASCADE delete |
| `role` | TEXT | `'user'` or `'assistant'` |
| `parts` | TEXT | JSON array of content parts |
| `model` / `provider` | TEXT | LLM info |
| `is_summary_message` | INTEGER | Flags context-compaction summaries |

### `files` — Versioned file snapshots
| Column | Type | Notes |
|---|---|---|
| `session_id` | TEXT FK | CASCADE delete |
| `path` | TEXT | |
| `content` | TEXT | Full file content |
| `version` | INTEGER | Auto-incrementing per path |
| | | `UNIQUE(path, session_id, version)` |

### `read_files` — Tracks read files per session
| Column | Type | Notes |
|---|---|---|
| `session_id` + `path` | Composite PK | |
| `read_at` | INTEGER | Last-read timestamp |

## Relationships
```
sessions (1) ──< (N) messages     [CASCADE]
sessions (1) ──< (N) files        [CASCADE]
sessions (1) ──< (N) read_files   [CASCADE]
sessions (1) ──< (N) sessions     [parent_session_id, sub-agent forking]
```

## 8 Migrations (goose, embedded, auto-applied at startup)
1. Initial schema (sessions, files, messages, triggers, indexes)
2. `summary_message_id` on sessions
3. `created_at` indexes
4. `provider` column on messages
5. `is_summary_message` flag
6. `todos` JSON column on sessions
7. `read_files` table
8. `inherited_message_count` on sessions

## Querier Interface — 40 Operations
Sessions CRUD, Messages CRUD (+ delete after, restore), Files CRUD (+ versioned), Read files, Stats/analytics (usage by day/model/hour, tool usage from JSON extraction).
