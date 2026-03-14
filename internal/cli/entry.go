package cli

import (
	"errors"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/spf13/cobra"
)

type entryOptions struct {
	json bool
}

func newEntryCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entry",
		Short: "Manage lightweight thinking units",
	}
	cmd.AddCommand(
		newEntryCreateCommand(opts),
		newEntryShowCommand(opts),
		newEntryListCommand(opts),
		newEntryArchiveCommand(opts),
		newEntryAttachCommand(opts),
		newEntryLinkCommand(opts),
		newEntryReviseCommand(opts),
	)
	return cmd
}

func newEntryCreateCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	var id, subject, state, meaning string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("entry id is required")
			}
			if subject == "" {
				return errors.New("entry subject is required")
			}
			entry := model.Entry{
				ID:        id,
				Subject:   subject,
				State:     state,
				Meaning:   meaning,
				CreatedAt: nowUTC(),
				UpdatedAt: nowUTC(),
			}
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			if err := store.SaveEntry(entry); err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "entry id")
	cmd.Flags().StringVar(&subject, "subject", "", "entry subject")
	cmd.Flags().StringVar(&state, "state", "", "entry state")
	cmd.Flags().StringVar(&meaning, "meaning", "", "entry meaning")
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryShowCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[0])
			if err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryListCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entries, err := store.ListEntries()
			if err != nil {
				return err
			}
			items := make([]entryListItem, 0, len(entries))
			for _, entry := range entries {
				items = append(items, entryListItem{
					ID:        entry.ID,
					Subject:   entry.Subject,
					State:     entry.State,
					UpdatedAt: entry.UpdatedAt,
				})
			}
			if entryOpts.json {
				return output(cmd, true, items, "")
			}
			if len(items) == 0 {
				return output(cmd, false, nil, "no entries\n")
			}
			text := ""
			for _, item := range items {
				text += formatEntryListItem(item) + "\n"
			}
			return output(cmd, false, nil, text)
		},
	}
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryReviseCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	var subject, state, meaning string

	cmd := &cobra.Command{
		Use:   "revise <id>",
		Short: "Revise an entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[0])
			if err != nil {
				return err
			}
			if subject != "" {
				entry.Subject = subject
			}
			if state != "" {
				entry.State = state
			}
			if meaning != "" {
				entry.Meaning = meaning
			}
			entry.UpdatedAt = nowUTC()
			if err := store.SaveEntry(entry); err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "update subject")
	cmd.Flags().StringVar(&state, "state", "", "update state")
	cmd.Flags().StringVar(&meaning, "meaning", "", "update meaning")
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryLinkCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	cmd := &cobra.Command{
		Use:   "link <id> <related-id> [more-related-ids...]",
		Short: "Add related context references to an entry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[0])
			if err != nil {
				return err
			}
			for _, related := range args[1:] {
				if !contains(entry.RelatedContext, related) {
					entry.RelatedContext = append(entry.RelatedContext, related)
				}
			}
			entry.UpdatedAt = nowUTC()
			if err := store.SaveEntry(entry); err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryArchiveCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	cmd := &cobra.Command{
		Use:   "archive <id>",
		Short: "Archive an entry without deleting it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[0])
			if err != nil {
				return err
			}
			entry.State = "archived"
			entry.UpdatedAt = nowUTC()
			if err := store.SaveEntry(entry); err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}

func newEntryAttachCommand(opts *rootOptions) *cobra.Command {
	entryOpts := &entryOptions{}
	var threadIDs []string
	var topicIDs []string

	cmd := &cobra.Command{
		Use:   "attach <id>",
		Short: "Attach an entry to thread support context and/or topic re-entry surfaces",
		Example: joinLines(
			"  sandnote entry attach en_auth --thread th_auth",
			"  sandnote entry attach en_auth --thread th_auth --topic tp_auth",
			"  sandnote entry attach en_auth --topic tp_auth --json",
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(threadIDs) == 0 && len(topicIDs) == 0 {
				return errors.New("attach requires at least one --thread or --topic target")
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			entry, err := store.LoadEntry(args[0])
			if err != nil {
				return err
			}

			now := nowUTC()
			for _, threadID := range threadIDs {
				thread, err := store.LoadThread(threadID)
				if err != nil {
					return err
				}
				if !contains(thread.SupportingIDs, entry.ID) {
					thread.SupportingIDs = append(thread.SupportingIDs, entry.ID)
					thread.UpdatedAt = now
					if err := store.SaveThread(thread); err != nil {
						return err
					}
				}
				if !contains(entry.RelatedContext, thread.ID) {
					entry.RelatedContext = append(entry.RelatedContext, thread.ID)
				}
			}

			for _, topicID := range topicIDs {
				topic, err := store.LoadTopic(topicID)
				if err != nil {
					return err
				}
				if !contains(topic.EntryIDs, entry.ID) {
					topic.EntryIDs = append(topic.EntryIDs, entry.ID)
					topic.UpdatedAt = now
					if err := store.SaveTopic(topic); err != nil {
						return err
					}
				}
				if !contains(entry.RelatedContext, topic.ID) {
					entry.RelatedContext = append(entry.RelatedContext, topic.ID)
				}
			}

			entry.UpdatedAt = now
			if err := store.SaveEntry(entry); err != nil {
				return err
			}
			return output(cmd, entryOpts.json, entry, formatEntry(entry))
		},
	}
	cmd.Flags().StringSliceVar(&threadIDs, "thread", nil, "attach the entry to one or more threads")
	cmd.Flags().StringSliceVar(&topicIDs, "topic", nil, "attach the entry to one or more topics")
	cmd.Flags().BoolVar(&entryOpts.json, "json", false, "output JSON")
	return cmd
}
