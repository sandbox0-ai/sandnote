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
		newEntryReviseCommand(opts),
	)
	addNotImplementedSubcommands(cmd, "link", "attach", "archive")
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
