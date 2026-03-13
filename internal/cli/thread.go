package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
	"github.com/spf13/cobra"
)

type threadOptions struct {
	json      bool
	vitality  string
	to        string
	workspace string
	topic     string
	query     string

	belief        string
	openEdge      string
	nextLean      string
	reentryAnchor string
}

type threadListItem struct {
	ID            string              `json:"id"`
	Question      string              `json:"question"`
	Vitality      model.VitalityState `json:"vitality"`
	WorkspaceID   string              `json:"workspace_id,omitempty"`
	TopicIDs      []string            `json:"topic_ids,omitempty"`
	UpdatedAt     time.Time           `json:"updated_at"`
	CurrentBelief string              `json:"current_belief,omitempty"`
	OpenEdge      string              `json:"open_edge,omitempty"`
}

type threadShowView struct {
	ID            string              `json:"id"`
	Question      string              `json:"question"`
	Vitality      model.VitalityState `json:"vitality"`
	CurrentBelief string              `json:"current_belief,omitempty"`
	OpenEdge      string              `json:"open_edge,omitempty"`
	WorkspaceID   string              `json:"workspace_id,omitempty"`
}

type threadResumeView struct {
	ID            string              `json:"id"`
	Question      string              `json:"question"`
	Vitality      model.VitalityState `json:"vitality"`
	CurrentBelief string              `json:"current_belief,omitempty"`
	OpenEdge      string              `json:"open_edge,omitempty"`
	NextLean      string              `json:"next_lean,omitempty"`
	ReentryAnchor string              `json:"reentry_anchor,omitempty"`
}

type threadInspectView struct {
	Thread            model.Thread  `json:"thread"`
	SupportingEntries []model.Entry `json:"supporting_entries,omitempty"`
}

func newThreadCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread",
		Short: "Work with continuable lines of thought",
	}

	cmd.AddCommand(
		newThreadCreateCommand(opts),
		newThreadFocusCommand(opts),
		newThreadShowCommand(opts),
		newThreadListCommand(opts),
		newThreadFrontierCommand(opts),
		newThreadResumeCommand(opts),
		newThreadInspectCommand(opts),
		newThreadEntriesCommand(opts),
		newThreadAttachCommand(opts),
		newThreadDetachCommand(opts),
		newThreadCheckpointCommand(opts),
		newThreadTransitionCommand(opts),
	)
	return cmd
}

func newThreadFocusCommand(opts *rootOptions) *cobra.Command {
	focusOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "focus <id>",
		Short: "Set the current focus thread for canonical CLI and REPL flows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			if err := updateActiveWorkSelection(store, thread.WorkspaceID, thread.ID); err != nil {
				return err
			}
			return output(cmd, focusOpts.json, threadShowView{
				ID:            thread.ID,
				Question:      thread.Question,
				Vitality:      thread.Vitality,
				CurrentBelief: thread.CurrentBelief,
				OpenEdge:      thread.OpenEdge,
				WorkspaceID:   thread.WorkspaceID,
			}, formatThreadShow(thread))
		},
	}
	cmd.Flags().BoolVar(&focusOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadFrontierCommand(opts *rootOptions) *cobra.Command {
	frontierOpts := &threadOptions{}
	var limit int

	cmd := &cobra.Command{
		Use:   "frontier",
		Short: "Show the ranked live frontier for continuable work",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			items, err := buildFrontier(store, frontierOpts.workspace)
			if err != nil {
				return err
			}
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}
			if frontierOpts.json {
				return output(cmd, true, items, "")
			}
			return output(cmd, false, nil, formatFrontier(items, limit))
		},
	}
	cmd.Flags().StringVar(&frontierOpts.workspace, "workspace", "", "prefer live threads from a workspace")
	cmd.Flags().IntVar(&limit, "limit", 5, "maximum number of frontier threads to show")
	cmd.Flags().BoolVar(&frontierOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadCreateCommand(opts *rootOptions) *cobra.Command {
	createOpts := &threadOptions{}
	var id string
	var question string
	var workspaceID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("thread id is required")
			}
			if question == "" {
				return errors.New("thread question is required")
			}
			now := time.Now().UTC()
			thread := model.Thread{
				ID:          id,
				Question:    question,
				Vitality:    model.VitalityLive,
				WorkspaceID: workspaceID,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			if err := store.SaveThread(thread); err != nil {
				return err
			}
			return output(cmd, createOpts.json, threadShowView{
				ID:          thread.ID,
				Question:    thread.Question,
				Vitality:    thread.Vitality,
				WorkspaceID: thread.WorkspaceID,
			}, formatThreadShow(thread))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "thread id")
	cmd.Flags().StringVar(&question, "question", "", "thread question")
	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().BoolVar(&createOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadShowCommand(opts *rootOptions) *cobra.Command {
	showOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show the canonical object overview of a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			view := threadShowView{
				ID:            thread.ID,
				Question:      thread.Question,
				Vitality:      thread.Vitality,
				CurrentBelief: thread.CurrentBelief,
				OpenEdge:      thread.OpenEdge,
				WorkspaceID:   thread.WorkspaceID,
			}
			return output(cmd, showOpts.json, view, formatThreadShow(thread))
		},
	}
	cmd.Flags().BoolVar(&showOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadListCommand(opts *rootOptions) *cobra.Command {
	listOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List threads",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validVitalityFilter(listOpts.vitality) {
				return fmt.Errorf("invalid vitality filter %q", listOpts.vitality)
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			derived, err := loadOrBuildIndex(store)
			if err != nil {
				return err
			}

			filtered := make([]threadListItem, 0, len(derived.Threads))
			for _, thread := range derived.Threads {
				if listOpts.vitality != "" && string(thread.Vitality) != listOpts.vitality {
					continue
				}
				if listOpts.workspace != "" && thread.WorkspaceID != listOpts.workspace {
					continue
				}
				if listOpts.topic != "" && !contains(thread.TopicIDs, listOpts.topic) {
					continue
				}
				if !matchesQuery(
					listOpts.query,
					thread.ID,
					thread.Question,
					thread.CurrentBelief,
					thread.OpenEdge,
					thread.WorkspaceID,
					strings.Join(thread.TopicIDs, " "),
				) {
					continue
				}
				filtered = append(filtered, threadListItem{
					ID:            thread.ID,
					Question:      thread.Question,
					Vitality:      thread.Vitality,
					WorkspaceID:   thread.WorkspaceID,
					TopicIDs:      thread.TopicIDs,
					UpdatedAt:     thread.UpdatedAt,
					CurrentBelief: thread.CurrentBelief,
					OpenEdge:      thread.OpenEdge,
				})
			}

			if listOpts.json {
				return output(cmd, true, filtered, "")
			}
			if len(filtered) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no threads")
				return nil
			}
			for _, item := range filtered {
				fmt.Fprintln(cmd.OutOrStdout(), formatThreadListItem(item))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&listOpts.vitality, "vitality", "", "filter by vitality state")
	cmd.Flags().StringVar(&listOpts.workspace, "workspace", "", "filter by workspace id")
	cmd.Flags().StringVar(&listOpts.topic, "topic", "", "filter by topic id")
	cmd.Flags().StringVar(&listOpts.query, "query", "", "filter by text query")
	cmd.Flags().BoolVar(&listOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadResumeCommand(opts *rootOptions) *cobra.Command {
	resumeOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "resume <id>",
		Short: "Restore the minimum continuation surface for a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			view := threadResumeView{
				ID:            thread.ID,
				Question:      thread.Question,
				Vitality:      thread.Vitality,
				CurrentBelief: thread.CurrentBelief,
				OpenEdge:      thread.OpenEdge,
				NextLean:      thread.NextLean,
				ReentryAnchor: thread.ReentryAnchor,
			}
			return output(cmd, resumeOpts.json, view, formatThreadResume(thread))
		},
	}
	cmd.Flags().BoolVar(&resumeOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadInspectCommand(opts *rootOptions) *cobra.Command {
	inspectOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a thread with supporting context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			entries, err := store.LoadEntries(thread.SupportingIDs)
			if err != nil {
				return err
			}

			view := threadInspectView{
				Thread:            thread,
				SupportingEntries: entries,
			}
			return output(cmd, inspectOpts.json, view, formatThreadInspect(thread, entries))
		},
	}
	cmd.Flags().BoolVar(&inspectOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadEntriesCommand(opts *rootOptions) *cobra.Command {
	entriesOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "entries <id>",
		Short: "List the entries currently supporting a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			entries, err := store.LoadEntries(thread.SupportingIDs)
			if err != nil {
				return err
			}
			if entriesOpts.json {
				return output(cmd, true, entries, "")
			}
			return output(cmd, false, nil, formatThreadEntries(thread, entries))
		},
	}
	cmd.Flags().BoolVar(&entriesOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadAttachCommand(opts *rootOptions) *cobra.Command {
	attachOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "attach <thread-id> <entry-id>",
		Short: "Attach an entry to a thread's supporting context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[1])
			if err != nil {
				return err
			}
			if !contains(thread.SupportingIDs, entry.ID) {
				thread.SupportingIDs = append(thread.SupportingIDs, entry.ID)
				thread.UpdatedAt = nowUTC()
				if err := store.SaveThread(thread); err != nil {
					return err
				}
			}
			entries, err := store.LoadEntries(thread.SupportingIDs)
			if err != nil {
				return err
			}
			if attachOpts.json {
				return output(cmd, true, threadInspectView{
					Thread:            thread,
					SupportingEntries: entries,
				}, "")
			}
			return output(cmd, false, nil, formatThreadEntries(thread, entries))
		},
	}
	cmd.Flags().BoolVar(&attachOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadDetachCommand(opts *rootOptions) *cobra.Command {
	detachOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "detach <thread-id> <entry-id>",
		Short: "Detach an entry from a thread's supporting context",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			nextSupporting := make([]string, 0, len(thread.SupportingIDs))
			for _, id := range thread.SupportingIDs {
				if id != args[1] {
					nextSupporting = append(nextSupporting, id)
				}
			}
			thread.SupportingIDs = nextSupporting
			thread.UpdatedAt = nowUTC()
			if err := store.SaveThread(thread); err != nil {
				return err
			}
			entries, err := store.LoadEntries(thread.SupportingIDs)
			if err != nil {
				return err
			}
			if detachOpts.json {
				return output(cmd, true, threadInspectView{
					Thread:            thread,
					SupportingEntries: entries,
				}, "")
			}
			return output(cmd, false, nil, formatThreadEntries(thread, entries))
		},
	}
	cmd.Flags().BoolVar(&detachOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadCheckpointCommand(opts *rootOptions) *cobra.Command {
	checkpointOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "checkpoint <id>",
		Short: "Leave behind a better stopping point for a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkpointOpts.belief == "" && checkpointOpts.openEdge == "" &&
				checkpointOpts.nextLean == "" && checkpointOpts.reentryAnchor == "" {
				return errors.New("checkpoint requires at least one of --belief, --open-edge, --next-lean, or --reentry-anchor")
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}

			if checkpointOpts.belief != "" {
				thread.CurrentBelief = checkpointOpts.belief
			}
			if checkpointOpts.openEdge != "" {
				thread.OpenEdge = checkpointOpts.openEdge
			}
			if checkpointOpts.nextLean != "" {
				thread.NextLean = checkpointOpts.nextLean
			}
			if checkpointOpts.reentryAnchor != "" {
				thread.ReentryAnchor = checkpointOpts.reentryAnchor
			}
			if err := validateCheckpointResult(thread); err != nil {
				return err
			}
			thread.UpdatedAt = time.Now().UTC()

			if err := store.SaveThread(thread); err != nil {
				return err
			}

			view := threadResumeView{
				ID:            thread.ID,
				Question:      thread.Question,
				Vitality:      thread.Vitality,
				CurrentBelief: thread.CurrentBelief,
				OpenEdge:      thread.OpenEdge,
				NextLean:      thread.NextLean,
				ReentryAnchor: thread.ReentryAnchor,
			}
			return output(cmd, checkpointOpts.json, view, formatThreadResume(thread))
		},
	}
	cmd.Flags().StringVar(&checkpointOpts.belief, "belief", "", "set current stance")
	cmd.Flags().StringVar(&checkpointOpts.openEdge, "open-edge", "", "set the open edge")
	cmd.Flags().StringVar(&checkpointOpts.nextLean, "next-lean", "", "set the likely next lean")
	cmd.Flags().StringVar(&checkpointOpts.reentryAnchor, "reentry-anchor", "", "set the re-entry anchor")
	cmd.Flags().BoolVar(&checkpointOpts.json, "json", false, "output JSON")
	return cmd
}

func newThreadTransitionCommand(opts *rootOptions) *cobra.Command {
	transitionOpts := &threadOptions{}
	cmd := &cobra.Command{
		Use:   "transition <id>",
		Short: "Change the vitality state of a thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if transitionOpts.to == "" {
				return errors.New("--to is required")
			}

			next := model.VitalityState(transitionOpts.to)
			if err := next.Validate(); err != nil {
				return err
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[0])
			if err != nil {
				return err
			}
			thread.Vitality = next
			thread.UpdatedAt = time.Now().UTC()

			if err := store.SaveThread(thread); err != nil {
				return err
			}
			if err := alignActiveSelectionAfterTransition(store, thread); err != nil {
				return err
			}

			view := threadShowView{
				ID:            thread.ID,
				Question:      thread.Question,
				Vitality:      thread.Vitality,
				CurrentBelief: thread.CurrentBelief,
				OpenEdge:      thread.OpenEdge,
				WorkspaceID:   thread.WorkspaceID,
			}
			return output(cmd, transitionOpts.json, view, formatThreadShow(thread))
		},
	}
	cmd.Flags().StringVar(&transitionOpts.to, "to", "", "target vitality state")
	cmd.Flags().BoolVar(&transitionOpts.json, "json", false, "output JSON")
	return cmd
}

func requireStore(root string) (*fsstore.Store, error) {
	store := fsstore.New(root)
	if !store.Initialized() {
		return nil, fmt.Errorf("sandnote store is not initialized at %s", root)
	}
	return store, nil
}

func output(cmd *cobra.Command, asJSON bool, value any, text string) error {
	if asJSON {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	if text != "" {
		fmt.Fprint(cmd.OutOrStdout(), text)
	}
	return nil
}

func formatThreadShow(thread model.Thread) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("thread %s", thread.ID))
	lines = append(lines, fmt.Sprintf("question: %s", thread.Question))
	lines = append(lines, fmt.Sprintf("vitality: %s", thread.Vitality))
	if thread.WorkspaceID != "" {
		lines = append(lines, fmt.Sprintf("workspace: %s", thread.WorkspaceID))
	}
	if thread.CurrentBelief != "" {
		lines = append(lines, fmt.Sprintf("belief: %s", thread.CurrentBelief))
	}
	if thread.OpenEdge != "" {
		lines = append(lines, fmt.Sprintf("edge: %s", thread.OpenEdge))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatThreadResume(thread model.Thread) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("resume %s", thread.ID))
	lines = append(lines, fmt.Sprintf("question: %s", thread.Question))
	if thread.CurrentBelief != "" {
		lines = append(lines, fmt.Sprintf("current belief: %s", thread.CurrentBelief))
	}
	if thread.OpenEdge != "" {
		lines = append(lines, fmt.Sprintf("open edge: %s", thread.OpenEdge))
	}
	if thread.NextLean != "" {
		lines = append(lines, fmt.Sprintf("next lean: %s", thread.NextLean))
	}
	if thread.ReentryAnchor != "" {
		lines = append(lines, fmt.Sprintf("re-entry anchor: %s", thread.ReentryAnchor))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatThreadInspect(thread model.Thread, entries []model.Entry) string {
	var lines []string
	lines = append(lines, formatThreadResume(thread))
	if len(entries) > 0 {
		lines = append(lines, "supporting entries:")
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("- %s: %s", entry.ID, entry.Subject))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatThreadEntries(thread model.Thread, entries []model.Entry) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("thread %s entries", thread.ID))
	if len(entries) == 0 {
		lines = append(lines, "no supporting entries")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("- %s: %s", entry.ID, entry.Subject))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatThreadListItem(item threadListItem) string {
	parts := []string{
		item.ID,
		"[" + string(item.Vitality) + "]",
		item.Question,
	}
	if item.WorkspaceID != "" {
		parts = append(parts, "workspace="+item.WorkspaceID)
	}
	if len(item.TopicIDs) > 0 {
		parts = append(parts, "topics="+strings.Join(item.TopicIDs, ","))
	}
	return strings.Join(parts, " ")
}

func validVitalityFilter(value string) bool {
	return value == "" || slices.Contains([]string{
		string(model.VitalityLive),
		string(model.VitalityDormant),
		string(model.VitalitySettled),
	}, value)
}

func validateCheckpointResult(thread model.Thread) error {
	if thread.Vitality != model.VitalityLive {
		return nil
	}

	missing := make([]string, 0, 2)
	if strings.TrimSpace(thread.OpenEdge) == "" {
		missing = append(missing, "open edge")
	}
	if strings.TrimSpace(thread.ReentryAnchor) == "" {
		missing = append(missing, "re-entry anchor")
	}
	if len(missing) > 0 {
		return fmt.Errorf("live thread checkpoints must leave a clear %s", joinCSV(missing))
	}
	return nil
}
