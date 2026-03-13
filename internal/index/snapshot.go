package index

import (
	"slices"
	"sort"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func Build(store *fsstore.Store) (fsstore.DerivedIndex, error) {
	threads, err := store.ListThreads()
	if err != nil {
		return fsstore.DerivedIndex{}, err
	}
	workspaces, err := store.ListWorkspaces()
	if err != nil {
		return fsstore.DerivedIndex{}, err
	}
	topics, err := store.ListTopics()
	if err != nil {
		return fsstore.DerivedIndex{}, err
	}

	threadTopics := make(map[string][]string, len(threads))
	for _, topic := range topics {
		for _, threadID := range topic.ThreadIDs {
			threadTopics[threadID] = append(threadTopics[threadID], topic.ID)
		}
	}
	for threadID := range threadTopics {
		sort.Strings(threadTopics[threadID])
	}

	derivedThreads := make([]fsstore.DerivedThreadRecord, 0, len(threads))
	for _, thread := range threads {
		derivedThreads = append(derivedThreads, fsstore.DerivedThreadRecord{
			ID:            thread.ID,
			Question:      thread.Question,
			Vitality:      thread.Vitality,
			WorkspaceID:   thread.WorkspaceID,
			TopicIDs:      slices.Clone(threadTopics[thread.ID]),
			SupportingIDs: slices.Clone(thread.SupportingIDs),
			CurrentBelief: thread.CurrentBelief,
			OpenEdge:      thread.OpenEdge,
			UpdatedAt:     thread.UpdatedAt,
		})
	}

	derivedWorkspaces := make([]fsstore.DerivedWorkspaceRecord, 0, len(workspaces))
	for _, workspace := range workspaces {
		threadCount := 0
		for _, thread := range threads {
			if thread.WorkspaceID == workspace.ID {
				threadCount++
			}
		}
		derivedWorkspaces = append(derivedWorkspaces, fsstore.DerivedWorkspaceRecord{
			ID:            workspace.ID,
			Name:          workspace.Name,
			FocusThreadID: workspace.FocusThreadID,
			ThreadCount:   threadCount,
			UpdatedAt:     workspace.UpdatedAt,
		})
	}

	derivedTopics := make([]fsstore.DerivedTopicRecord, 0, len(topics))
	for _, topic := range topics {
		derivedTopics = append(derivedTopics, fsstore.DerivedTopicRecord{
			ID:          topic.ID,
			Name:        topic.Name,
			Orientation: topic.Orientation,
			ThreadCount: len(topic.ThreadIDs),
			EntryCount:  len(topic.EntryIDs),
			UpdatedAt:   topic.UpdatedAt,
		})
	}

	return fsstore.DerivedIndex{
		GeneratedAt: time.Now().UTC(),
		Threads:     derivedThreads,
		Workspaces:  derivedWorkspaces,
		Topics:      derivedTopics,
	}, nil
}
