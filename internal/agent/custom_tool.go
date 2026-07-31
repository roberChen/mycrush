package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"charm.land/fantasy"

	"github.com/charmbracelet/crush/internal/agent/customtools"
	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
)

// customAgenticTool adapts a user-defined customtools.Definition into a
// fantasy.AgentTool. Each invocation spawns a fresh sub-agent (mirroring the
// agentic_fetch tool) whose system prompt, allowed tools, advertised skills,
// model tier and context-inheritance mode are all driven by the definition.
//
// Construction is intentionally cheap: the sub-agent, its models and its
// tools are built lazily inside Run. This keeps tool assembly recursion-free
// (buildTools may construct many customAgenticTool objects without each one
// eagerly building its own sub-agent) and lets a custom tool invoke another
// custom tool when explicitly allow-listed.
type customAgenticTool struct {
	def    *customtools.Definition
	coord  *coordinator
	info   fantasy.ToolInfo
	provOt fantasy.ProviderOptions
}

// newCustomAgenticTool builds the fantasy.AgentTool for a single definition.
// The JSON-schema parameters are derived from the definition so the model is
// told exactly how to call the tool.
func (c *coordinator) newCustomAgenticTool(def *customtools.Definition) fantasy.AgentTool {
	properties, required := def.BuildParameters()
	if required == nil {
		required = []string{}
	}
	return &customAgenticTool{
		def:   def.Clone(),
		coord: c,
		info: fantasy.ToolInfo{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  properties,
			Required:    required,
			Parallel:    def.IsParallel(),
		},
	}
}

func (t *customAgenticTool) Info() fantasy.ToolInfo { return t.info }

func (t *customAgenticTool) ProviderOptions() fantasy.ProviderOptions { return t.provOt }

func (t *customAgenticTool) SetProviderOptions(opts fantasy.ProviderOptions) {
	t.provOt = opts
}

func (t *customAgenticTool) Run(ctx context.Context, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	c := t.coord
	def := t.def

	sessionID := tools.GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, errors.New("session id missing from context")
	}
	agentMessageID := tools.GetMessageFromContext(ctx)
	if agentMessageID == "" {
		return fantasy.ToolResponse{}, errors.New("agent message id missing from context")
	}

	input, err := parseAndValidateInput(def, call.Input)
	if err != nil {
		return fantasy.NewTextErrorResponse(err.Error()), nil
	}

	userPrompt := renderToolInput(def, input)

	systemPrompt, err := t.buildSystemPrompt(ctx)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("build system prompt for %q: %w", def.Name, err)
	}

	large, small, err := c.buildAgentModels(ctx, true)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("build models for %q: %w", def.Name, err)
	}
	primary := large
	if def.EffectiveModel() == customtools.ModelChoiceSmall {
		primary = small
	}

	primaryProviderCfg, ok := c.cfg.Config().Providers.Get(primary.ModelCfg.Provider)
	if !ok {
		return fantasy.ToolResponse{}, fmt.Errorf("provider %q not configured for tool %q", primary.ModelCfg.Provider, def.Name)
	}

	subTools, err := c.buildTools(ctx, t.agentConfig(), true)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("build tools for %q: %w", def.Name, err)
	}

	subAgent := NewSessionAgent(SessionAgentOptions{
		LargeModel:           large,
		SmallModel:           small,
		SystemPromptPrefix:   primaryProviderCfg.SystemPromptPrefix,
		SystemPrompt:         systemPrompt,
		IsSubAgent:           true,
		DisableAutoSummarize: c.cfg.Config().Options.DisableAutoSummarize,
		IsYolo:               c.permissions.SkipRequests(),
		Sessions:             c.sessions,
		Messages:             c.messages,
		Tools:                subTools,
	})

	parentSessionID := sessionID
	return c.runSubAgent(ctx, subAgentParams{
		Agent:          subAgent,
		SessionID:      parentSessionID,
		AgentMessageID: agentMessageID,
		ToolCallID:     call.ID,
		Prompt:         userPrompt,
		SessionTitle:   fmt.Sprintf("Custom tool: %s", def.Name),
		SessionSetup: func(childSessionID string) {
			// Inherit the parent conversation so the sub-agent reasons over
			// the same context as the main agent. Best-effort: a copy failure
			// is logged but does not abort the run, since the sub-agent can
			// still complete from its system prompt + the tool input alone.
			if def.EffectiveContextMode() == customtools.ContextModeInherited {
				count, err := c.copyParentMessages(ctx, childSessionID, parentSessionID)
				if err != nil {
					slog.Warn("Failed to inherit parent context for custom tool",
						"tool", def.Name,
						"child_session", childSessionID,
						"parent_session", parentSessionID,
						"error", err,
					)
				} else if err := c.sessions.SetInheritedCount(ctx, childSessionID, int64(count)); err != nil {
					slog.Warn("Failed to persist inherited message count",
						"tool", def.Name,
						"child_session", childSessionID,
						"error", err,
					)
				}
			}
			// Sub-agent tool calls run non-interactively; auto-approve the
			// session so inner tool calls do not block waiting for a user.
			c.permissions.AutoApproveSession(childSessionID)
		},
	})
}

// agentConfig translates the definition into a config.Agent suitable for
// buildTools. MCP access defaults to "none" (matching the task agent) unless
// the definition explicitly opens it up.
func (t *customAgenticTool) agentConfig() config.Agent {
	def := t.def
	allowedMCP := def.AllowedMCP
	if allowedMCP == nil {
		allowedMCP = map[string][]string{}
	}
	modelType := config.SelectedModelTypeLarge
	if def.EffectiveModel() == customtools.ModelChoiceSmall {
		modelType = config.SelectedModelTypeSmall
	}
	return config.Agent{
		ID:           def.Name,
		Name:         def.Name,
		Description:  def.Description,
		Model:        modelType,
		AllowedTools: def.EffectiveAllowedTools(),
		AllowedMCP:   allowedMCP,
		ContextPaths: t.coord.cfg.Config().Options.ContextPaths,
	}
}

// buildSystemPrompt renders the definition's markdown body as a Go template
// (same data model as the coder/task prompts) and restricts the advertised
// skills to the definition's allow-list.
func (t *customAgenticTool) buildSystemPrompt(ctx context.Context) (string, error) {
	def := t.def
	opts := []prompt.Option{
		prompt.WithWorkingDir(t.coord.cfg.WorkingDir()),
	}
	if len(def.Skills) > 0 {
		opts = append(opts, prompt.WithAllowedSkills(def.Skills))
	}
	p, err := prompt.NewPrompt("custom_tool:"+def.Name, def.SystemPrompt, opts...)
	if err != nil {
		return "", err
	}
	model := config.SelectedModelTypeLarge
	if def.EffectiveModel() == customtools.ModelChoiceSmall {
		model = config.SelectedModelTypeSmall
	}
	selected := t.coord.cfg.Config().Models[model]
	return p.Build(ctx, selected.Provider, selected.Model, t.coord.cfg)
}

// parseAndValidateInput unmarshals the tool call input, fills in defaults for
// omitted optional parameters, and verifies that every required parameter is
// present.
func parseAndValidateInput(def *customtools.Definition, raw string) (map[string]any, error) {
	input := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	for _, p := range def.EffectiveParams() {
		_, present := input[p.Name]
		if !present {
			if p.Required {
				return nil, fmt.Errorf("parameter %q is required", p.Name)
			}
			if p.Default != nil {
				input[p.Name] = p.Default
			}
		}
	}
	return input, nil
}

// renderToolInput turns the validated input into the first user message for
// the sub-agent. It is intentionally human-readable so the sub-agent's system
// prompt can refer to parameter names directly; complex values are JSON
// encoded inline.
func renderToolInput(def *customtools.Definition, input map[string]any) string {
	var sb strings.Builder
	sb.WriteString("<tool_input>\n")
	for _, p := range def.EffectiveParams() {
		val, ok := input[p.Name]
		if !ok {
			continue
		}
		sb.WriteString(p.Name)
		sb.WriteString(": ")
		switch v := val.(type) {
		case string:
			sb.WriteString(v)
		case bool, float64, int, int64, json.Number:
			sb.WriteString(fmt.Sprint(v))
		default:
			encoded, err := json.Marshal(v)
			if err != nil {
				sb.WriteString(fmt.Sprint(v))
			} else {
				sb.Write(encoded)
			}
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("</tool_input>")
	return sb.String()
}

// copyParentMessages duplicates the parent session's message history into the
// child session, preserving roles, parts (including tool calls/results) and
// summary markers. It is the mechanism behind context_mode: inherited.
// Returns the number of messages copied; the caller persists this as an
// offset on the child session so the UI can skip inherited messages when
// rendering nested tool calls.
func (c *coordinator) copyParentMessages(ctx context.Context, childSessionID, parentSessionID string) (int, error) {
	msgs, err := c.messages.List(ctx, parentSessionID)
	if err != nil {
		return 0, fmt.Errorf("list parent messages: %w", err)
	}
	for _, m := range msgs {
		parts := slices.Clone(m.Parts)
		if _, err := c.messages.Create(ctx, childSessionID, message.CreateMessageParams{
			Role:             m.Role,
			Parts:            parts,
			Model:            m.Model,
			Provider:         m.Provider,
			IsSummaryMessage: m.IsSummaryMessage,
		}); err != nil {
			return 0, fmt.Errorf("copy message to child session: %w", err)
		}
	}
	return len(msgs), nil
}
