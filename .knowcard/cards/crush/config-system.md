---
id: 3eec6e1b5d09e71d0ae9abde64b0edcd
title: 'Configuration System: Loading, Providers, and Model Resolution'
keywords:
    - config
    - crush.json
    - providers
    - ConfigStore
    - model resolution
    - context paths
    - variable resolver
    - loading
summary: Crush's config system uses a ConfigStore with copy-on-write semantics. Config is loaded from multiple files (global, data, workspace) with deep-merge last-wins. Providers fetched from Catwalk catalog, models validated against catalog. Shell variable expansion supported for secrets.
created: 2026-07-10T19:24:44.811265858Z
updated: 2026-07-10T19:24:44.811265858Z
---

## Config Struct (`config.go`)

```go
type Config struct {
    Schema       string
    Models       map[SelectedModelType]SelectedModel   // "large" and "small"
    RecentModels map[SelectedModelType][]SelectedModel
    Providers    *csync.Map[string, ProviderConfig]
    MCP          MCPs        // map of MCP server configs
    LSP          LSPs        // map of LSP server configs
    Options      *Options    // context paths, debug, data dir, TUI, etc.
    Permissions  *Permissions // allowed_tools
    Tools        Tools       // ls/grep/glob settings
    Hooks        map[string][]HookConfig  // keyed by event name
    Agents       map[string]Agent         // runtime-built, not serialized
}
```

## Config Loading (`load.go`)

**`Load(workingDir, dataDir, debug)`** pipeline:
1. Migrate deprecated config keys
2. Discover config files in priority order (lowest → highest):
   - `~/.config/crush/crush.json` (global)
   - `~/.local/share/crush/crush.json` (global data)
   - `crush.json` / `.crush.json` (walk from cwd up to git root)
3. Deep-merge all configs (last-wins via `jsons.Merge`)
4. `.crush/crush.json` (workspace) merged last = highest priority
5. Set defaults (nil maps, data dir, context paths, skills paths)
6. Validate hooks (normalize event names, compile regexes)
7. Configure providers
8. Resolve selected models (validate against provider catalog)
9. Setup agents (coder + task)

## ConfigStore (`store.go`)

Copy-on-write config management:
- `configMu` (RWMutex) — guards config pointer
- `mu` (Mutex) — serializes disk writes
- `writeMu` (Mutex) — serializes in-memory production
- `cloneForWrite()` clones before mutation, atomic swap via `setConfig()`

## Provider Configuration (`provider.go`)

**`Providers(cfg)`** — fetches provider catalog (runs once via `sync.Once`):
- Concurrently fetches from Catwalk (`catwalk.charm.land/v2/providers`) + Hyper
- Caches to `$XDG_DATA_HOME/crush/providers.json`
- Hyper prepended if found

**`configureProviders()`** processes each provider:
- Known providers: merge user overrides on Catwalk defaults
- Custom providers: validate type, require BaseURL + model, auto-discover via `/v1/models`
- Provider-specific: Vertex AI (project+location), Azure (endpoint), Bedrock (AWS creds), Copilot (OAuth), Hyper (API key)

**`resolveSelectedModels()`** — validates user models against catalog, falls back to defaults, persists corrections.

## Key Options

| Option | Default | Description |
|---|---|---|
| `context_paths` | AGENTS.md, CRUSH.md, CLAUDE.md, etc. | Project context files |
| `data_directory` | `.crush` | SQLite DB + file storage |
| `debug` | false | Debug logging |
| `disabled_tools` | — | Tools to disable |
| `tui.compact_mode` | false | Compact TUI layout |
| `auto_lsp` | nil (auto) | Auto-start LSP servers |
| `notification_style` | — | Notification preferences |

## Variable Resolution (`resolve.go`)

`VariableResolver` interface:
- **`shellVariableResolver`** — expands `$VAR`, `${VAR}`, `${VAR:-default}`, `$(command)` via embedded shell. 5-min timeout.
- **`identityResolver`** — no-op, used in client/server mode

Errors sanitized — never exposes resolved secrets.
