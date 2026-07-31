package model

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent/customtools"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

type customToolStatusItem struct {
	icon  string
	name  string
	title string
	// description shows non-default model/context settings.
	description string
}

// customToolsInfo renders the custom agentic tools section showing loaded
// tool definitions and any that failed to load.
func (m *UI) customToolsInfo(width, maxItems int, isSection bool) string {
	t := m.com.Styles

	title := t.Resource.Heading.Render("Agent Tools")
	if isSection {
		title = common.Section(t, title, width)
	}

	items := m.customToolStatusItems()
	if len(items) == 0 {
		list := t.Resource.AdditionalText.Render("None")
		return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
	}

	list := customToolsList(t, items, width, maxItems)
	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

func (m *UI) customToolStatusItems() []customToolStatusItem {
	t := m.com.Styles
	var items []customToolStatusItem
	seen := make(map[string]bool, len(m.customToolDefs))

	for _, def := range m.customToolDefs {
		seen[def.Name] = true
		items = append(items, customToolStatusItem{
			icon:        t.Resource.OnlineIcon.String(),
			name:        def.Name,
			title:       t.Resource.Name.Render(def.Name),
			description: customToolDescription(t, def),
		})
	}

	for _, st := range m.customToolStates {
		if st.State != customtools.StateError || seen[st.Name] {
			continue
		}
		name := st.Name
		if name == "" {
			name = st.Path
		}
		items = append(items, customToolStatusItem{
			icon:        t.Resource.ErrorIcon.String(),
			name:        name,
			title:       t.Resource.Name.Render(name),
			description: t.Resource.StatusText.Render("error"),
		})
	}

	slices.SortStableFunc(items, func(a, b customToolStatusItem) int {
		return strings.Compare(a.name, b.name)
	})

	return items
}

// customToolDescription summarizes non-default model and context settings
// for a loaded custom agentic tool.
func customToolDescription(t *styles.Styles, def *customtools.Definition) string {
	var parts []string
	if def.EffectiveModel() == customtools.ModelChoiceSmall {
		parts = append(parts, "small model")
	}
	if def.EffectiveContextMode() == customtools.ContextModeInherited {
		parts = append(parts, "inherits context")
	}
	if len(parts) == 0 {
		return ""
	}
	return t.Resource.StatusText.Render(strings.Join(parts, " · "))
}

func customToolsList(t *styles.Styles, items []customToolStatusItem, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}

	if len(items) > maxItems {
		visibleItems := items[:maxItems-1]
		remaining := len(items) - (maxItems - 1)
		items = append(visibleItems, customToolStatusItem{
			name:  "more",
			title: t.Resource.AdditionalText.Render(fmt.Sprintf("…and %d more", remaining)),
		})
	}

	renderedItems := make([]string, 0, len(items))
	for _, item := range items {
		renderedItems = append(renderedItems, common.Status(t, common.StatusOpts{
			Icon:        item.icon,
			Title:       item.title,
			Description: item.description,
		}, width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedItems...)
}
