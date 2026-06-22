package agent

import (
	"testing"

	"github.com/charmbracelet/crush/internal/agent/customtools"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateInput_RequiredMissing(t *testing.T) {
	t.Parallel()

	def := &customtools.Definition{
		Params: []customtools.Param{{Name: "query", Type: "string", Required: true}},
	}
	_, err := parseAndValidateInput(def, `{}`)
	require.ErrorContains(t, err, `"query" is required`)
}

func TestParseAndValidateInput_DefaultsApplied(t *testing.T) {
	t.Parallel()

	def := &customtools.Definition{
		Params: []customtools.Param{
			{Name: "query", Type: "string", Required: true},
			{Name: "limit", Type: "integer", Default: 5},
		},
	}
	input, err := parseAndValidateInput(def, `{"query":"x"}`)
	require.NoError(t, err)
	require.Equal(t, "x", input["query"])
	require.EqualValues(t, 5, input["limit"])
}

func TestParseAndValidateInput_InvalidJSON(t *testing.T) {
	t.Parallel()

	def := &customtools.Definition{}
	_, err := parseAndValidateInput(def, `{not json`)
	require.ErrorContains(t, err, "invalid parameters")
}

func TestRenderToolInput(t *testing.T) {
	t.Parallel()

	def := &customtools.Definition{
		Params: []customtools.Param{
			{Name: "query", Type: "string"},
			{Name: "tags", Type: "array"},
		},
	}
	input := map[string]any{
		"query": "hello",
		"tags":  []any{"a", "b"},
	}
	out := renderToolInput(def, input)
	require.Contains(t, out, "query: hello")
	require.Contains(t, out, "tags: ")
	// Complex (array) values are JSON-encoded inline.
	require.Contains(t, out, `["a","b"]`)
}

func TestRenderToolInput_OmitsAbsent(t *testing.T) {
	t.Parallel()

	def := &customtools.Definition{
		Params: []customtools.Param{
			{Name: "a", Type: "string"},
			{Name: "b", Type: "string"},
		},
	}
	out := renderToolInput(def, map[string]any{"a": "1"})
	require.Contains(t, out, "a: 1")
	require.NotContains(t, out, "b:")
}
