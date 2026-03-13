package cli

import (
	"strings"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

func nowUTC() time.Time {
	return time.Now().UTC()
}

func joinLines(lines ...string) string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, "\n") + "\n"
}

type entryListItem struct {
	ID        string    `json:"id"`
	Subject   string    `json:"subject"`
	State     string    `json:"state,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func formatEntry(entry model.Entry) string {
	return joinLines(
		"entry "+entry.ID,
		"subject: "+entry.Subject,
		optionalLabel("state", entry.State),
		optionalLabel("meaning", entry.Meaning),
		optionalLabel("related", strings.Join(entry.RelatedContext, ", ")),
	)
}

func formatEntryListItem(item entryListItem) string {
	parts := []string{item.ID, item.Subject}
	if item.State != "" {
		parts = append(parts, "state="+item.State)
	}
	return strings.Join(parts, " ")
}

type workspaceListItem struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FocusThreadID string    `json:"focus_thread_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func formatWorkspace(workspace model.Workspace) string {
	return joinLines(
		"workspace "+workspace.ID,
		"name: "+workspace.Name,
		optionalLabel("focus thread", workspace.FocusThreadID),
		optionalLabel("threads", strings.Join(workspace.ThreadIDs, ", ")),
	)
}

func formatWorkspaceListItem(item workspaceListItem) string {
	parts := []string{item.ID, item.Name}
	if item.FocusThreadID != "" {
		parts = append(parts, "focus="+item.FocusThreadID)
	}
	return strings.Join(parts, " ")
}

type topicListItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Orientation string    `json:"orientation,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func formatTopic(topic model.Topic) string {
	return joinLines(
		"topic "+topic.ID,
		"name: "+topic.Name,
		optionalLabel("orientation", topic.Orientation),
		optionalLabel("entries", strings.Join(topic.EntryIDs, ", ")),
		optionalLabel("threads", strings.Join(topic.ThreadIDs, ", ")),
	)
}

func formatTopicListItem(item topicListItem) string {
	parts := []string{item.ID, item.Name}
	if item.Orientation != "" {
		parts = append(parts, "oriented")
	}
	return strings.Join(parts, " ")
}

func optionalLabel(name, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return name + ": " + value
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
