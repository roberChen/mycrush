// Package customtools provides discovery and parsing of user-defined
// "custom agentic tools".
//
// A custom agentic tool is a sub-agent that the main (coder) agent can
// invoke as a tool to offload a well-scoped task or to keep long-running
// exploration out of the main session's context. Each tool is defined by
// a directory containing a TOOL.md file whose YAML frontmatter describes
// the tool's metadata and input parameters and whose markdown body is the
// Go-text/template system prompt for the spawned sub-agent.
//
// The file layout intentionally mirrors the Agent Skills convention
// (a directory with a SKILL.md) so that a tool may co-locate helper
// scripts or reference assets next to its definition.
package customtools

import "sync"

// registeredToolNames holds the set of custom agentic tool names discovered
// at startup. The UI layer queries this via IsRegisteredToolName to decide
// whether a tool call should be rendered with the nested-tool-call view
// (live sub-agent progress) instead of a static "waiting for output" display.
var (
	registeredToolNames   = make(map[string]struct{})
	registeredToolNamesMu sync.RWMutex
)

// RegisterToolName records a custom agentic tool name so the UI layer can
// recognise it. Called by the coordinator during startup discovery.
func RegisterToolName(name string) {
	registeredToolNamesMu.Lock()
	registeredToolNames[name] = struct{}{}
	registeredToolNamesMu.Unlock()
}

// IsRegisteredToolName reports whether name was registered as a custom
// agentic tool. The UI uses this to decide whether to render a tool call
// with the nested sub-agent progress view.
func IsRegisteredToolName(name string) bool {
	registeredToolNamesMu.RLock()
	_, ok := registeredToolNames[name]
	registeredToolNamesMu.RUnlock()
	return ok
}

// ClearRegisteredToolNames removes all registered names. Intended for tests.
func ClearRegisteredToolNames() {
	registeredToolNamesMu.Lock()
	registeredToolNames = make(map[string]struct{})
	registeredToolNamesMu.Unlock()
}
