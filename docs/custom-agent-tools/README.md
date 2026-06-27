# Custom Agentic Tools

> [!NOTE]
> This document was designed for both humans and agents.

A **custom agentic tool** is a sub-agent that the main (coder) agent can invoke
as a tool, just like the built-in `agent` (task) and `agentic_fetch` tools.
You define one with a markdown file to offload a well-scoped task, or to keep a
long exploration out of the main session's context window.

Each tool is a **directory containing a `TOOL.md`** file. The directory name
must match the tool's `name`. The YAML frontmatter configures the tool's
metadata, input parameters and behavior; the markdown body is the Go
text/template system prompt for the spawned sub-agent (the same template data
model as the coder/task prompts: `{{.WorkingDir}}`, `{{.Platform}}`,
`{{.AvailSkillXML}}`, `{{.ContextFiles}}`, etc.).

### Hot Tool Facts

- A custom agentic tool is a directory with a `TOOL.md` file.
- The sub-agent runs non-interactively with its own tool set, system prompt and
  (optionally) skills.
- Tools are auto-registered for the top-level coder agent. A sub-agent can use
  another custom tool only if it lists the tool's name in `allowed_tools`.
- Custom tool names must be lowercase `snake_case` and must not collide with a
  built-in tool name (e.g. `bash`, `edit`, `agent`).

### Some things you can do with custom agentic tools:

- **Context offloading**: delegate a token-heavy investigation to a sub-agent
  that returns only a summary, keeping the main context small.
- **Focused workflows**: ship a purpose-built agent (e.g. "write a changelog
  from recent commits") with a tight tool set and prompt.
- **Inherited reasoning**: spawn a sub-agent that sees the whole conversation
  (`context_mode: inherited`) to answer a question with full context, without
  polluting the main transcript with its tool calls.

## Baby's First Tool

Create `.agents/agent_tools/summarize/TOOL.md`:

````markdown
---
name: summarize
description: Summarize the contents of a file in two sentences.
context_mode: none
allowed_tools:
  - view
params:
  - name: path
    description: Absolute path of the file to summarize.
    type: string
    required: true
---

You are a summarization sub-agent. Read the file given in the `path` input and
reply with a two-sentence summary. Do not include any other commentary.

Working directory: {{.WorkingDir}}
````

The coder agent can now call `summarize` like any other tool. Its sub-agent
only has access to the `view` tool and receives:

```text
<tool_input>
path: /absolute/path/to/file.go
</tool_input>
```

## Frontmatter Reference

| Field           | Type               | Default            | Description                                                                 |
| --------------- | ------------------ | ------------------ | --------------------------------------------------------------------------- |
| `name`          | string             | _(required)_       | Lowercase `snake_case` identifier; must match the directory name.          |
| `description`   | string             | _(required)_       | Shown to the model so it knows when to call the tool.                      |
| `context_mode`  | `none` \| `inherited` | `none`          | `inherited` copies the parent session's full message history into the sub-agent's session before running. |
| `allowed_tools` | string[]           | read-only set      | Tool whitelist for the sub-agent (`glob`, `grep`, `ls`, `sourcegraph`, `view`). |
| `allowed_mcp`   | map[string][]string | `{}` (none)       | MCP servers/tools the sub-agent may call. `nil` = all; empty map = none.   |
| `skills`        | string[]           | `[]` (none)        | Skills advertised to the sub-agent's system prompt.                        |
| `model`         | `large` \| `small` | `large`            | Which configured model tier the sub-agent uses.                            |
| `parallel`      | bool               | `true`             | Whether the tool may run in parallel with other tool calls.               |
| `params`        | Param[]            | a `prompt` string  | The tool's input schema (see below).                                       |

### `params`

When `params` is omitted the tool gets a single required `prompt` string
parameter (behaving like the built-in task tool). Each entry supports:

| Field         | Type    | Default   | Description                                                     |
| ------------- | ------- | --------- | --------------------------------------------------------------- |
| `name`        | string  | _(required)_ | Parameter identifier, sent to the sub-agent as `name: value`. |
| `description` | string  |           | JSON-schema description advertised to the model.                |
| `type`        | string  | `string`  | One of `string`, `integer`, `number`, `boolean`, `array`, `object`. |
| `required`    | bool    | `false`   | Whether the caller must supply the parameter.                   |
| `default`     | any     |           | Applied when an optional parameter is omitted.                  |
| `enum`        | any[]   |           | Restricts the value to the given choices.                       |

## Discovery Paths

Crush looks for `TOOL.md` files in (later paths override earlier ones with the
same name):

- `$CRUSH_AGENT_TOOLS_DIR` (if set)
- `~/.config/crush/agent_tools`
- `~/.agents/agent_tools`
- `<workdir>/.agents/agent_tools`
- `<workdir>/.crush/agent_tools`
- …and the same project subdirectories at the git work-tree root

Add your own paths in `crush.json`:

```json
{
  "options": {
    "custom_agent_tools_paths": ["./tools/agents"]
  }
}
```

## A Larger Example: Inherited-Context Question Answerer

````markdown
---
name: deep_answer
description: Answer a question using the full conversation context plus web search.
context_mode: inherited
allowed_tools:
  - glob
  - grep
  - view
  - fetch
  - agentic_fetch
skills:
  - jq
model: large
params:
  - name: question
    description: The question to answer.
    type: string
    required: true
---

You are a research sub-agent. You inherit the full conversation context.
Use the `question` input as your objective. Search the codebase and the web,
reason step by step, and return a concise, well-cited answer.

{{.AvailSkillXML}}
````

Because `context_mode: inherited`, the sub-agent sees everything the main agent
has discussed, but its (potentially long) tool results stay in the child
session and never bloat the main transcript — only the final answer is returned
to the coder agent.
