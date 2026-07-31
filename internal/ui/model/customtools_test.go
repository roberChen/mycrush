package model

import (
	"errors"
	"testing"

	"github.com/charmbracelet/crush/internal/agent/customtools"
	"github.com/charmbracelet/crush/internal/ui/common"
	uistyles "github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/stretchr/testify/require"
)

// TestCustomToolStatusItems verifies loaded definitions and failed loads are
// both reported in the sidebar section.
func TestCustomToolStatusItems(t *testing.T) {
	t.Parallel()

	st := uistyles.CharmtonePantera()
	ui := &UI{
		com: &common.Common{Styles: &st},
		customToolDefs: []*customtools.Definition{
			{Name: "code_review", Model: customtools.ModelChoiceSmall, ContextMode: customtools.ContextModeInherited},
		},
		customToolStates: []*customtools.State{
			{Name: "code_review", Path: "/tmp/code_review/TOOL.md", State: customtools.StateNormal},
			{Name: "", Path: "/tmp/broken_tool/TOOL.md", State: customtools.StateError, Err: errors.New("boom")},
		},
	}

	items := ui.customToolStatusItems()
	require.Len(t, items, 2)

	var hasCodeReview, hasBroken bool
	for _, item := range items {
		switch item.name {
		case "code_review":
			hasCodeReview = true
			require.Contains(t, item.description, "small model")
			require.Contains(t, item.description, "inherits context")
		case "/tmp/broken_tool/TOOL.md":
			hasBroken = true
			require.Contains(t, item.description, "error")
		}
	}
	require.True(t, hasCodeReview)
	require.True(t, hasBroken)
}

// TestCustomToolStatusItemsEmpty verifies the section renders "None" when no
// custom agentic tools are configured.
func TestCustomToolStatusItemsEmpty(t *testing.T) {
	t.Parallel()

	st := uistyles.CharmtonePantera()
	ui := &UI{
		com: &common.Common{Styles: &st},
	}

	items := ui.customToolStatusItems()
	require.Empty(t, items)

	out := ui.customToolsInfo(40, 10, false)
	require.Contains(t, out, "None")
}
