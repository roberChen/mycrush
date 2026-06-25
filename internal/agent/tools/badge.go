package tools

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/session"
)

//go:embed read_badge.md
var readBadgeDescription string

//go:embed update_badge.md
var updateBadgeDescription string

const (
	ReadBadgeToolName   = "read_badge"
	UpdateBadgeToolName = "update_badge"
)

// BadgeReadTracker enforces the read-before-update requirement for
// badge tools. The model must call read_badge before it can call
// update_badge within the same session.
type BadgeReadTracker struct {
	mu   sync.Mutex
	read map[string]bool
}

// NewBadgeReadTracker creates a new BadgeReadTracker.
func NewBadgeReadTracker() *BadgeReadTracker {
	return &BadgeReadTracker{read: make(map[string]bool)}
}

func (t *BadgeReadTracker) markRead(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.read[sessionID] = true
}

func (t *BadgeReadTracker) hasRead(sessionID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.read[sessionID]
}

// ReadBadgeParams has no parameters — the badge is read from the
// current session context.
type ReadBadgeParams struct{}

// UpdateBadgeParams contains the new badge content.
type UpdateBadgeParams struct {
	Badge string `json:"badge" description:"The new badge content. Set to empty string to clear the badge."`
}

// NewReadBadgeTool creates a tool that reads the current session badge.
func NewReadBadgeTool(sessions session.Service, tracker *BadgeReadTracker) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		ReadBadgeToolName,
		readBadgeDescription,
		func(ctx context.Context, _ ReadBadgeParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for reading badge")
			}

			sess, err := sessions.Get(ctx, sessionID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			tracker.markRead(sessionID)

			badge := sess.Badge
			if badge == "" {
				return fantasy.NewTextResponse("The session badge is currently empty. You can set it using the update_badge tool."), nil
			}

			return fantasy.NewTextResponse(fmt.Sprintf("Current session badge:\n\n%s", badge)), nil
		},
	)
}

// NewUpdateBadgeTool creates a tool that updates the session badge.
// The model must have called read_badge first within the same session.
func NewUpdateBadgeTool(sessions session.Service, tracker *BadgeReadTracker) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		UpdateBadgeToolName,
		updateBadgeDescription,
		func(ctx context.Context, params UpdateBadgeParams, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for updating badge")
			}

			if !tracker.hasRead(sessionID) {
				return fantasy.ToolResponse{}, fmt.Errorf("you must call read_badge before updating the badge")
			}

			sess, err := sessions.Get(ctx, sessionID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to get session: %w", err)
			}

			oldBadge := sess.Badge
			sess.Badge = params.Badge

			_, err = sessions.Save(ctx, sess)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("failed to save badge: %w", err)
			}

			if params.Badge == "" {
				return fantasy.NewTextResponse("Badge cleared successfully."), nil
			}

			if oldBadge == "" {
				return fantasy.NewTextResponse(fmt.Sprintf("Badge set successfully.\n\nNew badge:\n%s", params.Badge)), nil
			}

			return fantasy.NewTextResponse(fmt.Sprintf("Badge updated successfully.\n\nPrevious badge:\n%s\n\nNew badge:\n%s", oldBadge, params.Badge)), nil
		},
	)
}
