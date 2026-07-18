---
id: 8a34297906c2ad3d336280c1f58759b8
title: 'Agent System: Coordinator and SessionAgent'
keywords:
    - agent
    - coordinator
    - session agent
    - SessionAgent
    - fantasy
    - streaming
    - LLM
    - coder
    - task
    - provider options
    - loop detection
    - dispatch
    - queue
    - hookedTool
    - StopWhen
    - auto-summarize
    - PrepareStep
summary: 'The agent system consists of the Coordinator (manages named agents) and SessionAgent (runs LLM conversations per session). Run() has four phases: dispatch handoff (cancel-on-entry/queue-if-busy), setup, streaming loop via fantasy.Agent.Stream() with callbacks, and post-stream cleanup. Key features: async tool/prompt building, accept/cancel ordering via dispatchMu, loop detection (10-step window, 5 max repeats), context-window-based auto-summarize, hookedTool decorator for PreToolUse hooks, queue drain via recursive Run().'
created: 2026-07-10T19:39:05.609425394Z
updated: 2026-07-10T19:39:05.609425394Z
---

## The Coordinator (`internal/agent/coordinator.go`)

Top-level orchestrator managing one or more named agents. Currently has a single "coder" agent, designed for future multi-agent support.

### `Coordinator` interface
- `Run` / `RunAccepted` — Delegate to current agent
- `BeginAccepted`, `Cancel`, `CancelAll`, `IsSessionBusy`, queue management
- `UpdateModels` — Refresh model providers before each run
- `Summarize`, `GenerateTitle`, `Model`

### `run()` — shared implementation behind Run/RunAccepted
1. Waits for async agent readiness (`readyWg.Wait()`)
2. Refreshes models (`UpdateModels`)
3. Merges provider options (model → provider → catwalk defaults)
4. Handles unauthorized retries (`runWithUnauthorizedRetry` — refreshes OAuth/API key on 401, retries once)
5. Coalesces per-attempt `RunComplete` events — only final outcome reaches subscribers

### `buildAgent()`
Creates a `SessionAgent` with two async goroutines via `readyWg`:
- One builds the system prompt from Go template (`internal/agent/templates/*.md.tpl`)
- One builds the tool set via `buildTools()`

### `buildTools()` — Central tool registry
Constructs all tools, filters by agent's `AllowedTools` and `AllowedMCP`, wraps with `hookedTool` decorator (top-level only, runs PreToolUse hooks before permission checks), sorts alphabetically.

### `hookedTool` decorator (`internal/agent/hooked_tool.go`)
Wraps each tool at the coordinator level. Before executing a tool, it runs PreToolUse hooks (user-defined shell commands from `crush.json`). Hooks can: allow, deny, halt (stop the entire run), or rewrite tool input. Runs before permission checks.

## The SessionAgent (`internal/agent/agent.go`)

Core orchestration for AI conversations. ~780-line `Run()` method.

### Key struct: `sessionAgent`
Thread-safe via `csync` primitives:
- `largeModel`, `smallModel` — `csync.Value[Model]`
- `systemPrompt`, `systemPromptPrefix` — `csync.Value[string]`
- `tools` — `csync.Slice[fantasy.AgentTool]`
- `messageQueue` — `csync.Map[string, []SessionAgentCall]` (per-session FIFO)
- `activeRequests` — `csync.Map[string, context.CancelFunc]` (per-session cancel)
- `dispatchMu` — `csync.Map[string, *sync.Mutex]` (per-session mutexes serializing accept→cancel/queue transitions)
- `acceptedRuns`, `cancelMark`, `acceptSeqGen` — accept/cancel ordering machinery

### `Run()` method — Four phases

**Phase 1: Dispatch handoff (556-646)** — Determines whether to run, queue, or cancel:
- **Cancel-on-entry**: if a new run arrives while another is active in the same session, the active run is cancelled and the new one takes over (for interactive runs)
- **Queue-if-busy**: non-interactive runs are queued in `messageQueue`
- Uses `dispatchMu` per-session mutexes to serialize accept→cancel/queue transitions
- `acceptedRuns` and `cancelMark` track accept/cancel ordering

**Phase 2: Setup (648-798)** — Builds the fantasy agent:
- Constructs `fantasy.NewAgent(largeModel.Model, WithSystemPrompt, WithTools, WithUserAgent)`
- Loads history messages via `preparePrompt()` (handles image support, attachments)
- Creates user message in SQLite

**Phase 3: Streaming loop (800-1079)** — Calls `agent.Stream()` with all callbacks (see below)

**Phase 4: Post-stream (1081-1339)** — Cleanup:
- Error handling: persists final state, creates synthetic tool results for errors
- Auto-summarize if `shouldSummarize` flag was set by StopWhen
- Publishes `RunComplete` event
- **Queue drain via recursion**: if queued prompts exist, calls `a.Run(ctx, firstQueuedMessage)` recursively

### The streaming loop — callbacks

The actual loop is inside `fantasy.Agent.Stream()`. Each iteration ("step") is driven by callbacks:

**`PrepareStep`** (before each LLM call):
- Carries forward all prior messages, clears stale ProviderOptions
- Refreshes tool list (`a.tools.Copy()`) — MCP tools may have changed
- Drains queued follow-up prompts via `drainQueueForStep()`, folds them into message list
- Sets Anthropic cache control breakpoints on system prompt + last 2 messages
- Creates a NEW assistant message record in DB for this step

**Streaming deltas** (real-time, each writes to DB + pushes to UI):
- `OnReasoningStart/Delta/End` — thinking/reasoning tokens (handles provider-specific signatures)
- `OnTextDelta` — response text
- `OnToolInputStart` + `OnToolCall` — tool call detection (sanitizes input, handles hallucinated tool names)

**`OnToolResult`** — Persists tool results as `message.Tool` role in DB. Fantasy auto-includes them in next `PrepareStep`'s messages.

**`OnStepFinish`** — Maps finish reason (`Stop`→`EndTurn`, `ToolCalls`→`ToolUse`, `Length`→`MaxTokens`). If a tool result has `StopTurn` (hook halt / permission denial), forces `EndTurn`. Updates session usage (tokens, cost).

### Loop termination — `StopWhen`

Two stop conditions:
1. **Context window threshold**: if `contextWindow - (completionTokens + promptTokens) <= threshold` and auto-summarize is enabled → sets `shouldSummarize = true`, stops loop
   - Large context windows (>200K): threshold = 20,000 tokens
   - Smaller context windows: threshold = 20% of context window
2. **Loop detection**: delegates to `hasRepeatedToolCalls()` (see below)
3. **Built-in (fantasy)**: when `FinishReason` is `Stop` or `Length` (model stops requesting tools)

## Loop Detection (`internal/agent/loop_detection.go`)

Prevents infinite tool-call loops:

```go
const (
    loopDetectionWindowSize  = 10  // examine last 10 steps
    loopDetectionMaxRepeats  = 5   // max repetitions allowed
)

func hasRepeatedToolCalls(steps []fantasy.StepResult, windowSize, maxRepeats int) bool {
    // Hashes each step's tool calls via getToolInteractionSignature (toolName + input + output)
    // If any signature appears >maxRepeats in last windowSize steps → stop
}
```

## Two Built-in Agents
- **coder** — Large model, all tools (minus disabled), all MCPs, all context paths
- **task** — Large model, **read-only tools only** (glob, grep, ls, sourcegraph, view), **no MCPs**

## Data Flow
```
User Prompt
  → Coordinator.run() (refresh models, merge options, 401 retry, coalesce RunComplete)
  → SessionAgent.Run()
    → Phase 1: Dispatch (cancel-on-entry / queue-if-busy)
    → Phase 2: Build fantasy.Agent + create user message
    → Phase 3: agent.Stream() with callbacks
        ┌── Step loop ──────────────────────────────┐
        │ PrepareStep → LLM stream → OnToolCall     │
        │ → OnToolResult → OnStepFinish → StopWhen  │
        │ → (repeat if tools called & no stop)       │
        └────────────────────────────────────────────┘
    → Phase 4: Cleanup (errors, auto-summarize, queue drain via recursion)
```
