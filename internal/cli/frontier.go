package cli

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

type frontierItem struct {
	ID                   string    `json:"id"`
	Question             string    `json:"question"`
	WorkspaceID          string    `json:"workspace_id,omitempty"`
	CurrentBelief        string    `json:"current_belief,omitempty"`
	OpenEdge             string    `json:"open_edge,omitempty"`
	NextLean             string    `json:"next_lean,omitempty"`
	ReentryAnchor        string    `json:"reentry_anchor,omitempty"`
	ContinuationPressure int       `json:"continuation_pressure"`
	Reasons              []string  `json:"reasons,omitempty"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type frontierContext struct {
	targetWorkspace string
	focusThread     string
	workspaceFocus  map[string]string
}

func deriveFrontierContext(store *fsstore.Store, explicitWorkspace string) (frontierContext, error) {
	ctx := frontierContext{
		targetWorkspace: explicitWorkspace,
		workspaceFocus:  map[string]string{},
	}

	session, err := store.LoadREPLSession()
	if err != nil {
		return frontierContext{}, err
	}
	if ctx.targetWorkspace == "" {
		ctx.targetWorkspace = session.CurrentWorkspace
	}
	ctx.focusThread = session.FocusThread

	workspaces, err := store.ListWorkspaces()
	if err != nil {
		return frontierContext{}, err
	}
	for _, workspace := range workspaces {
		if workspace.FocusThreadID != "" {
			ctx.workspaceFocus[workspace.ID] = workspace.FocusThreadID
		}
	}

	return ctx, nil
}

func buildFrontier(store *fsstore.Store, explicitWorkspace string) ([]frontierItem, error) {
	derived, err := loadOrBuildIndex(store)
	if err != nil {
		return nil, err
	}
	ctx, err := deriveFrontierContext(store, explicitWorkspace)
	if err != nil {
		return nil, err
	}

	items := make([]frontierItem, 0, len(derived.Threads))
	for _, thread := range derived.Threads {
		if thread.Vitality != "live" {
			continue
		}
		if ctx.targetWorkspace != "" && thread.WorkspaceID != ctx.targetWorkspace {
			continue
		}
		score, reasons := continuationPressure(thread, ctx)
		items = append(items, frontierItem{
			ID:                   thread.ID,
			Question:             thread.Question,
			WorkspaceID:          thread.WorkspaceID,
			CurrentBelief:        thread.CurrentBelief,
			OpenEdge:             thread.OpenEdge,
			NextLean:             thread.NextLean,
			ReentryAnchor:        thread.ReentryAnchor,
			ContinuationPressure: score,
			Reasons:              reasons,
			UpdatedAt:            thread.UpdatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ContinuationPressure != items[j].ContinuationPressure {
			return items[i].ContinuationPressure > items[j].ContinuationPressure
		}
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func continuationPressure(thread fsstore.DerivedThreadRecord, ctx frontierContext) (int, []string) {
	score := 0
	reasons := make([]string, 0, 6)

	if thread.ID == ctx.focusThread {
		score += 100
		reasons = append(reasons, "focused thread")
	}
	if ctx.targetWorkspace != "" && thread.WorkspaceID == ctx.targetWorkspace {
		score += 40
		reasons = append(reasons, "workspace match")
	}
	if thread.WorkspaceID != "" && ctx.workspaceFocus[thread.WorkspaceID] == thread.ID {
		score += 25
		reasons = append(reasons, "workspace focus")
	}
	if strings.TrimSpace(thread.OpenEdge) != "" {
		score += 30
		reasons = append(reasons, "clear open edge")
	}
	if strings.TrimSpace(thread.ReentryAnchor) != "" {
		score += 20
		reasons = append(reasons, "re-entry anchor")
	}
	if strings.TrimSpace(thread.NextLean) != "" {
		score += 10
		reasons = append(reasons, "next lean")
	}
	if strings.TrimSpace(thread.CurrentBelief) != "" {
		score += 10
		reasons = append(reasons, "current belief")
	}

	return score, reasons
}

func bestFrontierItem(items []frontierItem) (frontierItem, error) {
	if len(items) == 0 {
		return frontierItem{}, errors.New("no live threads")
	}
	return items[0], nil
}

func formatFrontierItem(item frontierItem) string {
	parts := []string{
		item.ID,
		"pressure=" + fmtInt(item.ContinuationPressure),
		item.Question,
	}
	if item.WorkspaceID != "" {
		parts = append(parts, "workspace="+item.WorkspaceID)
	}
	if len(item.Reasons) > 0 {
		parts = append(parts, "reasons="+strings.Join(item.Reasons, ","))
	}
	return strings.Join(parts, " ")
}

func formatFrontier(items []frontierItem, limit int) string {
	if len(items) == 0 {
		return "no live threads\n"
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	lines := []string{"frontier"}
	for _, item := range items {
		lines = append(lines, "- "+formatFrontierItem(item))
	}
	return strings.Join(lines, "\n") + "\n"
}
