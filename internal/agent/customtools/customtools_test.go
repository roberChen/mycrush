package customtools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const validToolMD = `---
name: code_explorer
description: Explore a codebase and summarize how a feature works.
context_mode: inherited
allowed_tools:
  - glob
  - grep
  - view
skills:
  - jq
model: small
parallel: false
params:
  - name: query
    description: The feature to explore.
    type: string
    required: true
  - name: max_files
    description: Max files to inspect.
    type: integer
    required: false
    default: 20
---

You are a code exploration sub-agent.

Working directory: {{.WorkingDir}}
`

func TestParseContent_Valid(t *testing.T) {
	t.Parallel()

	def, err := ParseContent([]byte(validToolMD))
	require.NoError(t, err)
	require.Equal(t, "code_explorer", def.Name)
	require.Contains(t, def.Description, "Explore a codebase")
	require.Equal(t, ContextModeInherited, def.EffectiveContextMode())
	require.Equal(t, ModelChoiceSmall, def.EffectiveModel())
	require.False(t, def.IsParallel())
	require.Equal(t, []string{"glob", "grep", "view"}, def.EffectiveAllowedTools())
	require.Equal(t, []string{"jq"}, def.Skills)

	require.Len(t, def.Params, 2)
	require.Equal(t, "query", def.Params[0].Name)
	require.True(t, def.Params[0].Required)

	require.Equal(t, "You are a code exploration sub-agent.\n\nWorking directory: {{.WorkingDir}}", def.SystemPrompt)
}

func TestParseContent_NoFrontmatter(t *testing.T) {
	t.Parallel()

	_, err := ParseContent([]byte("# just a readme\nno frontmatter here"))
	require.ErrorContains(t, err, "no YAML frontmatter found")
}

func TestValidate_ReservedName(t *testing.T) {
	t.Parallel()

	def := &Definition{
		Name:        "bash",
		Description: "oops",
	}
	err := def.Validate()
	require.ErrorContains(t, err, "reserved")
}

func TestValidate_BadContextMode(t *testing.T) {
	t.Parallel()

	def := &Definition{
		Name:        "my_tool",
		Description: "ok",
		ContextMode: ContextMode("bogus"),
	}
	err := def.Validate()
	require.ErrorContains(t, err, "invalid context_mode")
}

func TestValidate_BadParamType(t *testing.T) {
	t.Parallel()

	def := &Definition{
		Name:        "my_tool",
		Description: "ok",
		Params:      []Param{{Name: "p", Type: "yaml"}},
	}
	err := def.Validate()
	require.ErrorContains(t, err, "invalid type")
}

func TestValidate_DuplicateParam(t *testing.T) {
	t.Parallel()

	def := &Definition{
		Name:        "my_tool",
		Description: "ok",
		Params: []Param{
			{Name: "p", Type: "string"},
			{Name: "p", Type: "string"},
		},
	}
	err := def.Validate()
	require.ErrorContains(t, err, "duplicate param name")
}

func TestDefaults_PromptFallback(t *testing.T) {
	t.Parallel()

	def := &Definition{Name: "my_tool", Description: "ok"}
	params := def.EffectiveParams()
	require.Len(t, params, 1)
	require.Equal(t, "prompt", params[0].Name)
	require.True(t, params[0].Required)

	require.Equal(t, DefaultAllowedTools, def.EffectiveAllowedTools())
	require.Equal(t, ModelChoiceLarge, def.EffectiveModel())
	require.Equal(t, ContextModeNone, def.EffectiveContextMode())
	require.True(t, def.IsParallel())
}

func TestBuildParameters(t *testing.T) {
	t.Parallel()

	def := &Definition{
		Name:        "my_tool",
		Description: "ok",
		Params: []Param{
			{Name: "query", Description: "q", Type: "string", Required: true},
			{Name: "limit", Type: "integer", Default: 10},
		},
	}
	props, required := def.BuildParameters()

	query, ok := props["query"].(ParamSchema)
	require.True(t, ok)
	require.Equal(t, "string", query["type"])
	require.Equal(t, "q", query["description"])

	limit, ok := props["limit"].(ParamSchema)
	require.True(t, ok)
	require.Equal(t, "integer", limit["type"])
	require.Equal(t, 10, limit["default"])

	require.Equal(t, []string{"query"}, required)
}

func TestBuildParameters_DefaultPromptParam(t *testing.T) {
	t.Parallel()

	def := &Definition{Name: "my_tool", Description: "ok"}
	props, required := def.BuildParameters()
	require.Contains(t, props, "prompt")
	require.Equal(t, []string{"prompt"}, required)
}

func TestParse_DirectoryNameMustMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	toolDir := filepath.Join(dir, "wrong_name")
	require.NoError(t, os.Mkdir(toolDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(toolDir, ToolFileName), []byte(validToolMD), 0o644))

	_, err := Parse(filepath.Join(toolDir, ToolFileName))
	require.ErrorContains(t, err, "must match directory")
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// A valid tool: dir name matches the "name" frontmatter.
	explorer := filepath.Join(root, "code_explorer")
	require.NoError(t, os.Mkdir(explorer, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(explorer, ToolFileName), []byte(validToolMD), 0o644))

	// A minimal tool using the default prompt param.
	minimal := filepath.Join(root, "summarizer")
	require.NoError(t, os.Mkdir(minimal, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(minimal, ToolFileName), []byte(`---
name: summarizer
description: Summarize something.
---
Do the thing.
`), 0o644))

	// An invalid tool (missing name): must not abort discovery.
	broken := filepath.Join(root, "broken")
	require.NoError(t, os.Mkdir(broken, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(broken, ToolFileName), []byte("---\ndescription: no name\n---\n"), 0o644))

	defs, states := Discover([]string{root})
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	require.ElementsMatch(t, []string{"code_explorer", "summarizer"}, names)

	// The broken tool should surface as an error state.
	var sawError bool
	for _, st := range states {
		if st.State == StateError {
			sawError = true
		}
	}
	require.True(t, sawError, "expected an error state for the invalid tool")
}

func TestDiscover_LastPathWins(t *testing.T) {
	t.Parallel()

	first := t.TempDir()
	second := t.TempDir()

	write := func(root, desc string) {
		dir := filepath.Join(root, "dupe")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ToolFileName), []byte("---\nname: dupe\ndescription: "+desc+"\n---\n"), 0o644))
	}
	write(first, "first")
	write(second, "second")

	defs, _ := Discover([]string{first, second})
	require.Len(t, defs, 1)
	require.Equal(t, "second", defs[0].Description)
}

func TestSplitFrontmatter_BOMAndCRLF(t *testing.T) {
	t.Parallel()

	// Leading UTF-8 BOM and Windows line endings must not break parsing.
	content := "\uFEFF---\r\nname: x\r\ndescription: y\r\n---\r\nbody\r\n"
	def, err := ParseContent([]byte(content))
	require.NoError(t, err)
	require.Equal(t, "x", def.Name)
	require.Equal(t, "body", def.SystemPrompt)
}
