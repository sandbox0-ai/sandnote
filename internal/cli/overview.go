package cli

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
	"github.com/spf13/cobra"
)

type overviewOptions struct {
	json          bool
	frontierLimit int
}

type overviewView struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Active      overviewActiveView  `json:"active"`
	Counts      overviewCountsView  `json:"counts"`
	Resume      overviewResumeView  `json:"resume"`
	Frontier    []frontierItem      `json:"frontier,omitempty"`
	Workspaces  []overviewWorkspace `json:"workspaces,omitempty"`
	Threads     []overviewThread    `json:"threads,omitempty"`
	Entries     []overviewEntry     `json:"entries,omitempty"`
	Topics      []overviewTopic     `json:"topics,omitempty"`
	Artifacts   []overviewArtifact  `json:"artifacts,omitempty"`
}

type overviewActiveView struct {
	WorkspaceID              string   `json:"workspace_id,omitempty"`
	FocusThreadID            string   `json:"focus_thread_id,omitempty"`
	InspectionScope          []string `json:"inspection_scope,omitempty"`
	PendingCheckpointContext string   `json:"pending_checkpoint_context,omitempty"`
}

type overviewCountsView struct {
	Workspaces     int `json:"workspaces"`
	Threads        int `json:"threads"`
	LiveThreads    int `json:"live_threads"`
	DormantThreads int `json:"dormant_threads"`
	SettledThreads int `json:"settled_threads"`
	Entries        int `json:"entries"`
	Topics         int `json:"topics"`
	Artifacts      int `json:"artifacts"`
}

type overviewResumeView struct {
	Status       string `json:"status"`
	NextThreadID string `json:"next_thread_id,omitempty"`
}

type overviewWorkspace struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FocusThreadID string    `json:"focus_thread_id,omitempty"`
	ThreadCount   int       `json:"thread_count"`
	Active        bool      `json:"active,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type overviewThread struct {
	ID                 string              `json:"id"`
	Question           string              `json:"question"`
	Vitality           model.VitalityState `json:"vitality"`
	WorkspaceID        string              `json:"workspace_id,omitempty"`
	TopicIDs           []string            `json:"topic_ids,omitempty"`
	SupportingEntryIDs []string            `json:"supporting_entry_ids,omitempty"`
	ArtifactIDs        []string            `json:"artifact_ids,omitempty"`
	CurrentBelief      string              `json:"current_belief,omitempty"`
	OpenEdge           string              `json:"open_edge,omitempty"`
	NextLean           string              `json:"next_lean,omitempty"`
	ReentryAnchor      string              `json:"reentry_anchor,omitempty"`
	CheckpointGaps     []string            `json:"checkpoint_gaps,omitempty"`
	Focused            bool                `json:"focused,omitempty"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type overviewEntry struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	State       string    `json:"state,omitempty"`
	Meaning     string    `json:"meaning,omitempty"`
	ThreadIDs   []string  `json:"thread_ids,omitempty"`
	TopicIDs    []string  `json:"topic_ids,omitempty"`
	ArtifactIDs []string  `json:"artifact_ids,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type overviewTopic struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Orientation string    `json:"orientation,omitempty"`
	ThreadCount int       `json:"thread_count"`
	EntryCount  int       `json:"entry_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type overviewArtifact struct {
	ID               string                   `json:"id"`
	Kind             string                   `json:"kind"`
	SourceRef        string                   `json:"source_ref"`
	IngestMode       model.ArtifactIngestMode `json:"ingest_mode"`
	RelatedEntryIDs  []string                 `json:"related_entry_ids,omitempty"`
	RelatedThreadIDs []string                 `json:"related_thread_ids,omitempty"`
	ActiveThreadIDs  []string                 `json:"active_thread_ids,omitempty"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

func newOverviewCommand(opts *rootOptions) *cobra.Command {
	overviewOpts := &overviewOptions{}
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Show an agent-oriented overview of the current notebook",
		Example: joinLines(
			"  sandnote overview",
			"  sandnote overview --json",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			overview, err := buildOverview(store, overviewOpts.frontierLimit)
			if err != nil {
				return err
			}
			return output(cmd, overviewOpts.json, overview, formatOverviewAgent(overview))
		},
	}
	cmd.Flags().BoolVar(&overviewOpts.json, "json", false, "output JSON")
	cmd.Flags().IntVar(&overviewOpts.frontierLimit, "frontier-limit", 5, "maximum number of frontier threads to include")
	return cmd
}

func buildOverview(store *fsstore.Store, frontierLimit int) (overviewView, error) {
	derived, err := loadOrBuildIndex(store)
	if err != nil {
		return overviewView{}, err
	}
	session, err := store.LoadREPLSession()
	if err != nil {
		return overviewView{}, err
	}
	entries, err := store.ListEntries()
	if err != nil {
		return overviewView{}, err
	}
	topics, err := store.ListTopics()
	if err != nil {
		return overviewView{}, err
	}
	frontier, err := buildFrontier(store, "")
	if err != nil {
		return overviewView{}, err
	}
	if frontierLimit > 0 && len(frontier) > frontierLimit {
		frontier = frontier[:frontierLimit]
	}

	artifactSet := map[string]fsstore.DerivedArtifactRecord{}
	for _, artifact := range derived.Artifacts {
		artifactSet[artifact.ID] = artifact
	}

	entryThreadIDs := map[string][]string{}
	threadArtifactIDs := map[string][]string{}
	threadTopicIDs := map[string][]string{}
	artifactEntryIDs := map[string][]string{}
	artifactThreadIDs := map[string][]string{}

	entryArtifactIDs := map[string][]string{}
	for _, entry := range entries {
		for _, related := range entry.RelatedContext {
			if _, ok := artifactSet[related]; !ok {
				continue
			}
			if !contains(entryArtifactIDs[entry.ID], related) {
				entryArtifactIDs[entry.ID] = append(entryArtifactIDs[entry.ID], related)
			}
			if !contains(artifactEntryIDs[related], entry.ID) {
				artifactEntryIDs[related] = append(artifactEntryIDs[related], entry.ID)
			}
		}
	}

	for _, thread := range derived.Threads {
		threadTopicIDs[thread.ID] = slices.Clone(thread.TopicIDs)
		for _, entryID := range thread.SupportingIDs {
			if !contains(entryThreadIDs[entryID], thread.ID) {
				entryThreadIDs[entryID] = append(entryThreadIDs[entryID], thread.ID)
			}
			for _, artifactID := range entryArtifactIDs[entryID] {
				if !contains(threadArtifactIDs[thread.ID], artifactID) {
					threadArtifactIDs[thread.ID] = append(threadArtifactIDs[thread.ID], artifactID)
				}
				if !contains(artifactThreadIDs[artifactID], thread.ID) {
					artifactThreadIDs[artifactID] = append(artifactThreadIDs[artifactID], thread.ID)
				}
			}
		}
	}

	entryTopicIDs := map[string][]string{}
	for _, topic := range topics {
		for _, entryID := range topic.EntryIDs {
			if !contains(entryTopicIDs[entryID], topic.ID) {
				entryTopicIDs[entryID] = append(entryTopicIDs[entryID], topic.ID)
			}
		}
	}

	overview := overviewView{
		GeneratedAt: derived.GeneratedAt,
		Active: overviewActiveView{
			WorkspaceID:              session.CurrentWorkspace,
			FocusThreadID:            session.FocusThread,
			InspectionScope:          slices.Clone(session.InspectionScope),
			PendingCheckpointContext: session.PendingCheckpointContext,
		},
		Frontier: frontier,
	}
	if len(frontier) == 0 {
		overview.Resume.Status = "no_live_threads"
	} else {
		overview.Resume.Status = "ready"
		overview.Resume.NextThreadID = frontier[0].ID
	}

	for _, workspace := range derived.Workspaces {
		overview.Workspaces = append(overview.Workspaces, overviewWorkspace{
			ID:            workspace.ID,
			Name:          workspace.Name,
			FocusThreadID: workspace.FocusThreadID,
			ThreadCount:   workspace.ThreadCount,
			Active:        workspace.ID == session.CurrentWorkspace,
			UpdatedAt:     workspace.UpdatedAt,
		})
	}

	for _, thread := range derived.Threads {
		overview.Threads = append(overview.Threads, overviewThread{
			ID:                 thread.ID,
			Question:           thread.Question,
			Vitality:           thread.Vitality,
			WorkspaceID:        thread.WorkspaceID,
			TopicIDs:           slices.Clone(threadTopicIDs[thread.ID]),
			SupportingEntryIDs: slices.Clone(thread.SupportingIDs),
			ArtifactIDs:        slices.Clone(threadArtifactIDs[thread.ID]),
			CurrentBelief:      thread.CurrentBelief,
			OpenEdge:           thread.OpenEdge,
			NextLean:           thread.NextLean,
			ReentryAnchor:      thread.ReentryAnchor,
			CheckpointGaps:     checkpointGaps(thread),
			Focused:            thread.ID == session.FocusThread,
			UpdatedAt:          thread.UpdatedAt,
		})
		switch thread.Vitality {
		case model.VitalityLive:
			overview.Counts.LiveThreads++
		case model.VitalityDormant:
			overview.Counts.DormantThreads++
		case model.VitalitySettled:
			overview.Counts.SettledThreads++
		}
	}

	for _, entry := range entries {
		overview.Entries = append(overview.Entries, overviewEntry{
			ID:          entry.ID,
			Subject:     entry.Subject,
			State:       entry.State,
			Meaning:     entry.Meaning,
			ThreadIDs:   slices.Clone(entryThreadIDs[entry.ID]),
			TopicIDs:    slices.Clone(entryTopicIDs[entry.ID]),
			ArtifactIDs: slices.Clone(entryArtifactIDs[entry.ID]),
			UpdatedAt:   entry.UpdatedAt,
		})
	}

	for _, topic := range derived.Topics {
		overview.Topics = append(overview.Topics, overviewTopic{
			ID:          topic.ID,
			Name:        topic.Name,
			Orientation: topic.Orientation,
			ThreadCount: topic.ThreadCount,
			EntryCount:  topic.EntryCount,
			UpdatedAt:   topic.UpdatedAt,
		})
	}

	for _, artifact := range derived.Artifacts {
		activeThreadIDs := make([]string, 0, len(artifactThreadIDs[artifact.ID]))
		for _, threadID := range artifactThreadIDs[artifact.ID] {
			if contains(frontierThreadIDs(frontier), threadID) {
				activeThreadIDs = append(activeThreadIDs, threadID)
			}
		}
		overview.Artifacts = append(overview.Artifacts, overviewArtifact{
			ID:               artifact.ID,
			Kind:             artifact.Kind,
			SourceRef:        artifact.SourceRef,
			IngestMode:       artifact.IngestMode,
			RelatedEntryIDs:  slices.Clone(artifactEntryIDs[artifact.ID]),
			RelatedThreadIDs: slices.Clone(artifactThreadIDs[artifact.ID]),
			ActiveThreadIDs:  activeThreadIDs,
			UpdatedAt:        artifact.UpdatedAt,
		})
	}

	overview.Counts.Workspaces = len(overview.Workspaces)
	overview.Counts.Threads = len(overview.Threads)
	overview.Counts.Entries = len(overview.Entries)
	overview.Counts.Topics = len(overview.Topics)
	overview.Counts.Artifacts = len(overview.Artifacts)
	return overview, nil
}

func checkpointGaps(thread fsstore.DerivedThreadRecord) []string {
	gaps := []string{}
	if strings.TrimSpace(thread.OpenEdge) == "" {
		gaps = append(gaps, "missing_open_edge")
	}
	if strings.TrimSpace(thread.ReentryAnchor) == "" {
		gaps = append(gaps, "missing_reentry_anchor")
	}
	return gaps
}

func frontierThreadIDs(items []frontierItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func formatOverviewAgent(overview overviewView) string {
	lines := []string{"overview"}
	if overview.Active.WorkspaceID != "" {
		lines = append(lines, "active workspace: "+overview.Active.WorkspaceID)
	}
	if overview.Active.FocusThreadID != "" {
		lines = append(lines, "focus thread: "+overview.Active.FocusThreadID)
	} else if overview.Active.WorkspaceID != "" {
		lines = append(lines, "focus thread: none")
	}
	lines = append(lines, "resume status: "+overview.Resume.Status)
	if overview.Resume.NextThreadID != "" {
		lines = append(lines, "next thread: "+overview.Resume.NextThreadID)
	}
	lines = append(lines, fmt.Sprintf(
		"counts: workspaces=%d threads=%d live=%d dormant=%d settled=%d entries=%d topics=%d artifacts=%d",
		overview.Counts.Workspaces,
		overview.Counts.Threads,
		overview.Counts.LiveThreads,
		overview.Counts.DormantThreads,
		overview.Counts.SettledThreads,
		overview.Counts.Entries,
		overview.Counts.Topics,
		overview.Counts.Artifacts,
	))
	lines = append(lines, "workspaces:")
	if len(overview.Workspaces) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, workspace := range overview.Workspaces {
			parts := []string{workspace.ID, workspace.Name}
			if workspace.Active {
				parts = append(parts, "active")
			}
			if workspace.FocusThreadID != "" {
				parts = append(parts, "focus="+workspace.FocusThreadID)
			}
			parts = append(parts, "threads="+fmtInt(workspace.ThreadCount))
			lines = append(lines, "- "+strings.Join(parts, " "))
		}
	}
	lines = append(lines, "frontier:")
	if len(overview.Frontier) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, item := range overview.Frontier {
			lines = append(lines, "- "+formatFrontierItem(item))
		}
	}

	lines = append(lines, "threads:")
	if len(overview.Threads) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, thread := range overview.Threads {
			parts := []string{thread.ID, string(thread.Vitality)}
			if thread.WorkspaceID != "" {
				parts = append(parts, "workspace="+thread.WorkspaceID)
			}
			if len(thread.TopicIDs) > 0 {
				parts = append(parts, "topics="+joinCSV(thread.TopicIDs))
			}
			if len(thread.SupportingEntryIDs) > 0 {
				parts = append(parts, "entries="+joinCSV(thread.SupportingEntryIDs))
			}
			if len(thread.ArtifactIDs) > 0 {
				parts = append(parts, "artifacts="+joinCSV(thread.ArtifactIDs))
			}
			if len(thread.CheckpointGaps) > 0 {
				parts = append(parts, "gaps="+joinCSV(thread.CheckpointGaps))
			}
			if thread.ReentryAnchor != "" {
				parts = append(parts, "anchor="+thread.ReentryAnchor)
			}
			parts = append(parts, thread.Question)
			lines = append(lines, "- "+strings.Join(parts, " "))
		}
	}

	lines = append(lines, "entries:")
	if len(overview.Entries) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, entry := range overview.Entries {
			parts := []string{entry.ID, entry.Subject}
			if len(entry.ThreadIDs) > 0 {
				parts = append(parts, "threads="+joinCSV(entry.ThreadIDs))
			}
			if len(entry.TopicIDs) > 0 {
				parts = append(parts, "topics="+joinCSV(entry.TopicIDs))
			}
			if len(entry.ArtifactIDs) > 0 {
				parts = append(parts, "artifacts="+joinCSV(entry.ArtifactIDs))
			}
			lines = append(lines, "- "+strings.Join(parts, " "))
		}
	}

	lines = append(lines, "topics:")
	if len(overview.Topics) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, topic := range overview.Topics {
			parts := []string{topic.ID, topic.Name}
			if topic.Orientation != "" {
				parts = append(parts, "orientation="+topic.Orientation)
			}
			parts = append(parts, "threads="+fmtInt(topic.ThreadCount))
			parts = append(parts, "entries="+fmtInt(topic.EntryCount))
			lines = append(lines, "- "+strings.Join(parts, " "))
		}
	}

	lines = append(lines, "artifacts:")
	if len(overview.Artifacts) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, artifact := range overview.Artifacts {
			parts := []string{artifact.ID, artifact.Kind, "mode=" + string(artifact.IngestMode)}
			if len(artifact.RelatedThreadIDs) > 0 {
				parts = append(parts, "threads="+joinCSV(artifact.RelatedThreadIDs))
			}
			if artifact.SourceRef != "" {
				parts = append(parts, artifact.SourceRef)
			}
			lines = append(lines, "- "+strings.Join(parts, " "))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}
