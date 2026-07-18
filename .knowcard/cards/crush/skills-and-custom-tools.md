---
id: ad14438cfb3b281dfd996b13e824451b
title: Skills System and Custom Agentic Tools
keywords:
    - skills
    - SKILL.md
    - skill discovery
    - custom tools
    - TOOL.md
    - builtin skills
    - agentic tools
    - sub-agent
summary: Skills are SKILL.md files with YAML frontmatter + instructions. Builtin (embedded) + user skills discovered from multiple paths. User overrides builtin. Custom agentic tools (TOOL.md) spawn sub-agents with scoped tools/skills and Go-template system prompts.
created: 2026-07-10T19:25:32.595948778Z
updated: 2026-07-10T19:25:32.595948778Z
---

## Skills System (`internal/skills/`)

### Skill File Format (`SKILL.md`)

YAML frontmatter + markdown body:

```markdown
---
name: crush-config                    # required, kebab-case, must match dir name
description: Use when ...             # required, max 1024 chars
user-invocable: true                  # optional (default false)
disable-model-invocation: false       # optional - exclude from system prompt
license: MIT                          # optional
compatibility: ">=1.0"                # optional
metadata:                             # optional free-form
  author: charmbracelet
---

Markdown body becomes skill.Instructions
```

### Discovery (Two Layers)

**Builtin skills** (`embed.go`):
- `//go:embed builtin/*` embeds all builtin SKILL.md into binary
- Paths prefixed with `crush://skills/`
- `DiscoverBuiltinWithStates()` walks embed.FS

**User skills** (`skills.go`):
- `DiscoverWithStates(paths)` walks directories via `fastwalk`
- Looks for files named `SKILL.md`
- Deduplicates by path, validates each

**Unified:** `DiscoverFromConfig()` in `manager.go`:
1. Discover builtin → discover user
2. `Deduplicate()` — **user skills override builtins** with same name (last-wins)
3. `Filter()` — removes skills in `cfg.DisabledSkills`

### Manager (`manager.go`)
```go
type Manager struct {
    allSkills     []*Skill       // pre-filter
    activeSkills  []*Skill       // post-filter
    states        []*SkillState  // discovery diagnostics
    broker        *pubsub.Broker[Event]
}
```
Thread-safe, workspace-scoped. Publishes state changes via pubsub.

### Integration Points

1. **System prompt** — `skills.ToPromptXML()` → `<available_skills>` block in system prompt
2. **View tool** — Builtin skills read from embed FS (`crush://skills/` prefix); user skills bypass permission + size limits
3. **Usage tracking** — `Tracker` records which skills loaded during session
4. **`crush_info` tool** — Reports all/active skills + tracker state

### Skill Paths
- **Global:** `~/.config/crush/skills`, `~/.config/agents/skills`, `~/.agents/skills`, `~/.claude/skills`
- **Project:** `.agents/skills`, `.crush/skills`, `.claude/skills`, `.cursor/skills`

## Custom Agentic Tools (`internal/agent/customtools/`)

### TOOL.md Format
```yaml
---
name: my_tool                      # snake_case, must match dir name
description: "What this tool does"
context_mode: none                 # or "inherited" (copy parent messages)
allowed_tools: [glob, grep, view]  # default: read-only set
allowed_mcp: {}                    # nil=all, {}=none
skills: [crush-config]             # scoped skills for sub-agent
model: large                       # or "small"
parallel: true
params:                            # JSON-schema input params (default: single "prompt")
  - name: query
    type: string
    required: true
---
Markdown body = Go template system prompt for spawned sub-agent
```

### Integration (`internal/agent/custom_tool.go`)
- `coordinator.buildTools()` iterates custom tools, wraps each as `fantasy.AgentTool`
- `customAgenticTool.Run()` lazily builds fresh sub-agent with definition's system prompt
- `context_mode: inherited` copies parent session's message history
- Sub-agent sessions auto-approved for permissions
- `RegisterToolName()` called at startup for UI rendering
