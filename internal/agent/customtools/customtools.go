package customtools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/charlievieth/fastwalk"
	"gopkg.in/yaml.v3"
)

// ToolFileName is the marker file that identifies a custom agentic tool
// directory.
const ToolFileName = "TOOL.md"

// ContextMode controls how much of the parent (main) session's context the
// spawned sub-agent starts with.
type ContextMode string

const (
	// ContextModeNone (the default) starts the sub-agent with an empty
	// session — only the tool input is passed as the first user message.
	ContextModeNone ContextMode = "none"
	// ContextModeInherited copies the parent session's message history into
	// the sub-agent's session before running, giving the sub-agent the full
	// conversation context the main agent has accumulated.
	ContextModeInherited ContextMode = "inherited"
)

// ModelChoice selects which configured model the sub-agent uses.
type ModelChoice string

const (
	ModelChoiceLarge ModelChoice = "large"
	ModelChoiceSmall ModelChoice = "small"
)

// namePattern restricts tool names to lowercase snake_case so they fit
// alongside the built-in tool identifiers (e.g. "agentic_fetch").
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// reservedNames are built-in tool identifiers that a custom tool must not
// shadow, to avoid ambiguity in tool routing and permission checks.
var reservedNames = map[string]bool{
	"agent": true, "bash": true, "crush_info": true, "crush_logs": true,
	"job_output": true, "job_kill": true, "download": true, "edit": true,
	"multiedit": true, "lsp_diagnostics": true, "lsp_references": true,
	"lsp_restart": true, "fetch": true, "agentic_fetch": true, "glob": true,
	"grep": true, "ls": true, "sourcegraph": true, "todos": true,
	"view": true, "write": true, "list_mcp_resources": true,
	"read_mcp_resource": true,
}

// Param describes a single input parameter of a custom agentic tool. It is
// the user-facing equivalent of a struct field tag on a built-in tool: it
// is turned into a JSON-schema property advertised to the model and is
// validated before the sub-agent runs.
type Param struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// Type is the JSON-schema type of the parameter. Defaults to "string".
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
	// Default is applied when the caller omits the parameter. It is only
	// meaningful for non-required parameters.
	Default any   `yaml:"default"`
	Enum    []any `yaml:"enum"`
}

// Definition is the parsed configuration for one custom agentic tool.
type Definition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// ContextMode controls parent-context inheritance. Defaults to "none".
	ContextMode ContextMode `yaml:"context_mode"`
	// AllowedTools is the whitelist of tool names the sub-agent may use.
	// When empty, DefaultAllowedTools is used.
	AllowedTools []string `yaml:"allowed_tools"`
	// AllowedMCP restricts which MCP servers (and optionally which of their
	// tools) the sub-agent may call. Semantics match config.Agent.AllowedMCP:
	// nil = all MCPs, empty map = no MCPs, a map entry with nil/empty list =
	// all tools from that server.
	AllowedMCP map[string][]string `yaml:"allowed_mcp"`
	// Skills is the allow-list of skill names to advertise to the sub-agent.
	// When empty, no skills are advertised.
	Skills []string `yaml:"skills"`
	// Model selects the configured model tier. Defaults to "large".
	Model ModelChoice `yaml:"model"`
	// Parallel marks the tool as safe to run alongside other tools. Defaults
	// to true.
	Parallel *bool `yaml:"parallel"`
	// Params defines the tool's input schema. When empty, a single required
	// "prompt" string parameter is assumed so a minimal tool behaves like the
	// built-in task tool.
	Params []Param `yaml:"params"`

	// SystemPrompt is the markdown body of TOOL.md, used as the Go template
	// for the sub-agent's system prompt.
	SystemPrompt string `yaml:"-"`

	// Path is the directory containing TOOL.md.
	Path string `yaml:"-"`
	// FilePath is the absolute path to TOOL.md.
	FilePath string `yaml:"-"`
}

// DefaultAllowedTools is the read-only tool set a sub-agent gets when the
// definition does not specify AllowedTools. It mirrors the built-in task
// agent's read-only subset so an unconfigured custom tool cannot mutate the
// workspace by accident.
var DefaultAllowedTools = []string{"glob", "grep", "ls", "sourcegraph", "view"}

// EffectiveAllowedTools returns the tool whitelist, applying the default
// when none was configured.
func (d *Definition) EffectiveAllowedTools() []string {
	if len(d.AllowedTools) > 0 {
		return d.AllowedTools
	}
	return slices.Clone(DefaultAllowedTools)
}

// EffectiveParams returns the input parameters, defaulting to a single
// required "prompt" string parameter when none were configured.
func (d *Definition) EffectiveParams() []Param {
	if len(d.Params) > 0 {
		return d.Params
	}
	return []Param{{
		Name:        "prompt",
		Description: "The task for the agent to perform.",
		Type:        "string",
		Required:    true,
	}}
}

// EffectiveModel returns the configured model tier, defaulting to large.
func (d *Definition) EffectiveModel() ModelChoice {
	if d.Model == "" {
		return ModelChoiceLarge
	}
	return d.Model
}

// EffectiveContextMode returns the configured context mode, defaulting to none.
func (d *Definition) EffectiveContextMode() ContextMode {
	if d.ContextMode == "" {
		return ContextModeNone
	}
	return d.ContextMode
}

// IsParallel reports whether the tool may run in parallel with others.
func (d *Definition) IsParallel() bool {
	if d.Parallel == nil {
		return true
	}
	return *d.Parallel
}

// Validate checks the definition for spec compliance.
func (d *Definition) Validate() error {
	var errs []error

	if d.Name == "" {
		errs = append(errs, errors.New("name is required"))
	} else {
		if !namePattern.MatchString(d.Name) {
			errs = append(errs, fmt.Errorf("name %q must be lowercase snake_case (letters, digits, underscore, starting with a letter)", d.Name))
		}
		if reservedNames[d.Name] {
			errs = append(errs, fmt.Errorf("name %q is reserved by a built-in tool", d.Name))
		}
		if d.Path != "" && !strings.EqualFold(filepath.Base(d.Path), d.Name) {
			errs = append(errs, fmt.Errorf("name %q must match directory %q", d.Name, filepath.Base(d.Path)))
		}
	}

	if d.Description == "" {
		errs = append(errs, errors.New("description is required"))
	}

	switch d.ContextMode {
	case "", ContextModeNone, ContextModeInherited:
	default:
		errs = append(errs, fmt.Errorf("invalid context_mode %q (want none or inherited)", d.ContextMode))
	}

	switch d.Model {
	case "", ModelChoiceLarge, ModelChoiceSmall:
	default:
		errs = append(errs, fmt.Errorf("invalid model %q (want large or small)", d.Model))
	}

	seenParam := make(map[string]bool)
	for i, p := range d.Params {
		if p.Name == "" {
			errs = append(errs, fmt.Errorf("params[%d]: name is required", i))
			continue
		}
		if seenParam[p.Name] {
			errs = append(errs, fmt.Errorf("params[%d]: duplicate param name %q", i, p.Name))
		}
		seenParam[p.Name] = true
		switch p.Type {
		case "", "string", "integer", "number", "boolean", "array", "object":
		default:
			errs = append(errs, fmt.Errorf("params[%d] %q: invalid type %q", i, p.Name, p.Type))
		}
	}

	return errors.Join(errs...)
}

// Parse reads and validates a TOOL.md file from disk.
func Parse(path string) (*Definition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	def, err := ParseContent(content)
	if err != nil {
		return nil, err
	}
	def.FilePath = path
	def.Path = filepath.Dir(path)
	if err := def.Validate(); err != nil {
		return nil, err
	}
	return def, nil
}

// ParseContent parses a TOOL.md from raw bytes without validating or setting
// filesystem paths.
func ParseContent(content []byte) (*Definition, error) {
	frontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}
	var def Definition
	if err := yaml.Unmarshal([]byte(frontmatter), &def); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	def.SystemPrompt = strings.TrimSpace(body)
	return &def, nil
}

// splitFrontmatter extracts YAML frontmatter and the body from markdown. It
// mirrors the skills package's parser so TOOL.md files authored alongside
// SKILL.md files behave consistently (BOM stripping, CRLF normalization).
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	if start >= len(lines) || strings.TrimSpace(lines[start]) != "---" {
		return "", "", errors.New("no YAML frontmatter found")
	}
	endOffset := slices.IndexFunc(lines[start+1:], func(line string) bool {
		return strings.TrimSpace(line) == "---"
	})
	if endOffset == -1 {
		return "", "", errors.New("unclosed frontmatter")
	}
	end := start + 1 + endOffset
	frontmatter = strings.Join(lines[start+1:end], "\n")
	body = strings.Join(lines[end+1:], "\n")
	return frontmatter, body, nil
}

// DiscoveryState describes the outcome of parsing a single candidate file.
type DiscoveryState string

const (
	StateNormal DiscoveryState = "normal"
	StateError  DiscoveryState = "error"
)

// State records the parse result for a single candidate path, useful for
// diagnostics and UI reporting.
type State struct {
	Name  string
	Path  string
	State DiscoveryState
	Err   error
}

// Discover walks the given directories looking for TOOL.md files and returns
// the valid definitions. Errors for individual files are logged and reported
// via the returned states but do not abort discovery.
func Discover(paths []string) ([]*Definition, []*State) {
	var defs []*Definition
	var states []*State
	var mu sync.Mutex
	seen := make(map[string]bool)
	addState := func(name, path string, state DiscoveryState, err error) {
		mu.Lock()
		states = append(states, &State{Name: name, Path: path, State: state, Err: err})
		mu.Unlock()
	}

	for _, base := range paths {
		conf := fastwalk.Config{
			Follow:  true,
			ToSlash: fastwalk.DefaultToSlash(),
		}
		err := fastwalk.Walk(&conf, base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				addState("", path, StateError, err)
				return nil
			}
			if d.IsDir() || d.Name() != ToolFileName {
				return nil
			}
			mu.Lock()
			if seen[strings.ToLower(path)] {
				mu.Unlock()
				return nil
			}
			seen[strings.ToLower(path)] = true
			mu.Unlock()

			def, parseErr := Parse(path)
			if parseErr != nil {
				addState("", path, StateError, parseErr)
				return nil
			}
			mu.Lock()
			defs = append(defs, def)
			mu.Unlock()
			addState(def.Name, path, StateNormal, nil)
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			addState("", base, StateError, err)
		}
	}

	// Last-wins dedup by name (later paths override earlier ones), mirroring
	// the skills precedence model.
	defs = Deduplicate(defs)

	slices.SortStableFunc(defs, func(a, b *Definition) int {
		return strings.Compare(a.Name, b.Name)
	})
	return defs, states
}

// Deduplicate keeps the last definition for each name, preserving the input
// order of the survivors.
func Deduplicate(defs []*Definition) []*Definition {
	last := make(map[string]int, len(defs))
	for i, d := range defs {
		last[d.Name] = i
	}
	out := defs[:0]
	for i, d := range defs {
		if last[d.Name] == i {
			out = append(out, d)
		}
	}
	return out
}
