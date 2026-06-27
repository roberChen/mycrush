---
name: custom-agentic-tools
description:
  Use when the user needs to create, edit, or manage custom agentic tools for
  Crush. Covers tool definition authoring, registration, description files,
  parameters, and wiring into the agent tool system.
---

# Custom Agentic Tools

Crush supports user-defined custom agentic tools that extend the built-in tool
set. These tools are discovered at startup and registered alongside built-in
tools like `bash`, `edit`, `view`, etc.

## What Are Custom Agentic Tools

Custom agentic tools are Go functions that the model can invoke during a session.
Each tool has:
- A unique name (e.g., `my_tool`)
- A JSON-schema parameter struct
- A description file (`.md` or `.md.tpl`) explaining what the tool does
- A handler function that executes the tool logic

## Tool Definition File

Create a `.go` file that defines your tool. Follow this pattern:

```go
package customtools

import (
    "context"
    _ "embed"

    "charm.land/fantasy"
)

//go:embed my_tool.md
var myToolDescription string

const MyToolName = "my_tool"

type MyToolParams struct {
    Path string `json:"path" description:"The file path to process"`
}

func NewMyTool() fantasy.AgentTool {
    return fantasy.NewAgentTool(
        MyToolName,
        myToolDescription,
        func(ctx context.Context, params MyToolParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
            // Your tool logic here
            result := "Processed: " + params.Path
            return fantasy.NewTextResponse(result), nil
        },
    )
}
```

## Description File

Create a matching `.md` file (e.g., `my_tool.md`) in the same directory:

```markdown
# my_tool

Process a file at the given path and return a summary.

## Parameters

- `path` (string): The absolute or relative path to the file.

## Usage

The model calls this tool when it needs to analyze a file's contents and
produce a structured summary.
```

## Registration

Tools are discovered at startup via `customAgentToolsPaths` in `crush.json`:

```json
{
  "options": {
    "custom_agent_tools_paths": ["./my-tools/"]
  }
}
```

Crush scans each path for `.go` files, compiles them, and registers valid
tool definitions. The tool name must be unique and not conflict with built-in
tools.

## Parameter Struct Rules

- Use `json:"field_name"` tags for JSON serialization.
- Add `description:"..."` tags for schema documentation.
- Use `omitempty` for optional fields.
- Supported types: `string`, `int`, `int64`, `float64`, `bool`, arrays, and
  nested structs.

## Handler Function

The handler receives:
- `ctx context.Context` — for cancellation and timeouts
- `params MyToolParams` — parsed and validated parameters
- `call fantasy.ToolCall` — raw tool call metadata (ID, name, etc.)

Return:
- `fantasy.ToolResponse` — the tool result (text, error, or metadata)
- `error` — if the tool fails to execute

## Testing Custom Tools

Write unit tests for your tool handler:

```go
func TestMyTool(t *testing.T) {
    tool := NewMyTool()
    resp, err := tool.Handler()(context.Background(), MyToolParams{Path: "/tmp/test.txt"}, fantasy.ToolCall{})
    require.NoError(t, err)
    require.Contains(t, resp.Text(), "Processed")
}
```

## Best Practices

- Keep tool descriptions concise but complete — the model reads them to decide
  when to call the tool.
- Validate all parameters before use; return clear error messages.
- Use `slog` for logging, not `fmt.Print`.
- Handle context cancellation gracefully.
- Avoid side effects in parameter validation — do work in the handler.
- For file I/O, respect the working directory and use absolute paths when
  possible.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Tool not appearing in tool list | Path not in `custom_agent_tools_paths` | Add to `crush.json` |
| "duplicate tool name" error | Name conflicts with built-in | Rename the tool |
| Schema validation fails | Missing `json` tags on params | Add tags to all fields |
| Model never calls tool | Description too vague | Improve description with examples |
