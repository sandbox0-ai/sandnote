package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/index"
	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
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
	ThreadCount   int       `json:"thread_count"`
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
	if item.ThreadCount > 0 {
		parts = append(parts, "threads="+fmtInt(item.ThreadCount))
	}
	return strings.Join(parts, " ")
}

type topicListItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Orientation string    `json:"orientation,omitempty"`
	ThreadCount int       `json:"thread_count"`
	EntryCount  int       `json:"entry_count"`
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
	if item.ThreadCount > 0 {
		parts = append(parts, "threads="+fmtInt(item.ThreadCount))
	}
	if item.EntryCount > 0 {
		parts = append(parts, "entries="+fmtInt(item.EntryCount))
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

func fmtInt(value int) string {
	return strconv.Itoa(value)
}

func matchesQuery(query string, values ...string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return true
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), q) {
			return true
		}
	}
	return false
}

func loadOrBuildIndex(store *fsstore.Store) (fsstore.DerivedIndex, error) {
	derived, err := store.LoadDerivedIndex()
	if err == nil {
		return derived, nil
	}
	derived, err = index.Build(store)
	if err != nil {
		return fsstore.DerivedIndex{}, err
	}
	if err := store.SaveDerivedIndex(derived); err != nil {
		return fsstore.DerivedIndex{}, err
	}
	return derived, nil
}
