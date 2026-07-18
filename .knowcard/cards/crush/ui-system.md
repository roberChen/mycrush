---
id: f6a09844e1d14e0106f4010e9e15335d
title: 'UI System: Bubble Tea v2 TUI and Styling'
keywords:
    - UI
    - TUI
    - Bubble Tea
    - rendering
    - styling
    - themes
    - dialog
    - list
    - chat
    - compact mode
summary: 'TUI built on Bubble Tea v2 with single centralized UI model. Sub-components are imperative structs. Hybrid rendering via Ultraviolet screen buffers. Versioned render cache for lazy lists. Stacked dialog system. Three-layer styling: Styles struct, theme factories, token-driven quickStyle builder.'
created: 2026-07-10T19:26:13.535005313Z
updated: 2026-07-10T19:26:13.535005313Z
---

## TUI Architecture (`internal/ui/`)

Built on **Bubble Tea v2** with a hybrid rendering pipeline (Ultraviolet screen-buffer + string-based).

### Core Design Principles (from `internal/ui/AGENTS.md`)

1. **Single Model** — `UI` is the sole Bubble Tea model. Sub-components are stateful structs with imperative methods, NOT Elm-architecture participants.
2. **No IO in Update** — All expensive work goes through `tea.Cmd`; state mutations only in `Update` loop.
3. **ANSI-safe string ops** — Always use `charmbracelet/x/ansi`.

### Rendering Pipeline
`View()` → `uv.ScreenBuffer` → `Draw()` paints sub-regions → `canvas.Render()` flattens to string.

Layout is `uiLayout` struct computing rectangles for: header, main (chat), editor, sidebar, pills, status.

### Event Subscription Flow
```
Backend (pubsub Broker) → proto events
  → ClientWorkspace.Subscribe(program) [goroutine]
  → translateEvent() converts proto → domain types
  → program.Send(pubsub.Event[T])
  → UI.Update(msg) → switch on event type
```

### UI States
`uiOnboarding` → `uiInitialize` → `uiLanding` → `uiChat`

### Focus Routing
`uiFocusEditor` → textarea, `uiFocusMain` → chat list, `uiFocusNone` → neither

### Key Packages

| Package | Role |
|---------|------|
| `model/` | The `UI` struct — top-level model, owns all state, message routing |
| `chat/` | Chat message items + per-tool renderers (bash, file, search, mcp, etc.) |
| `list/` | Generic lazy-rendered scrollable list with versioned render cache |
| `dialog/` | Stacked dialog system (Overlay push/pop). Implementations: models, sessions, permissions, API key, OAuth, filepicker, quit, etc. |
| `common/` | `Common` struct (workspace + styles), layout helpers, markdown/diff rendering |
| `completions/` | Autocomplete popup (`@` mentions) |
| `attachments/` | File attachment management |
| `styles/` | Centralized style definitions, themes, color tokens |
| `diffview/` | Unified + split diff rendering with syntax highlighting |
| `anim/` | Animated spinner |
| `image/` | Terminal image rendering (Kitty graphics protocol) |
| `notification/` | Desktop notifications (bell, OSC, native) |

## Styling System (`internal/ui/styles/`)

Three layers:

| File | Role |
|------|------|
| `styles.go` | `Styles` struct with nested groups (Header, Messages, Tool, Pills, Editor, Sidebar, Diff). No hardcoded colors. |
| `themes.go` | Theme factories. `ThemeForProvider()` maps provider → theme. Themes: `CharmtonePantera` (default), `HypercrushObsidiana` (Hyper). |
| `quickstyle.go` | `quickStyle(opts)` builder takes a palette (~40 color roles) and generates full `Styles` struct. Fully token-driven. |

### Adding a New Theme
1. Add function in `themes.go` calling `quickStyle` with palette
2. Apply theme-specific overrides for colors that don't fit token model
3. Wire into `ThemeForProvider`

### Key Pattern: Versioned Render Cache
List items embed `*Versioned`, bump version on mutation. List freezes `Finished()` items for zero-cost re-render. Only visible items rendered.
