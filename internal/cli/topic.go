package cli

import (
	"errors"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/spf13/cobra"
)

type topicOptions struct {
	json bool
}

func newTopicCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Manage durable re-entry surfaces",
	}
	cmd.AddCommand(
		newTopicCreateCommand(opts),
		newTopicShowCommand(opts),
		newTopicListCommand(opts),
		newTopicOrientCommand(opts),
		newTopicPromoteCommand(opts),
	)
	addNotImplementedSubcommands(cmd, "entries", "threads")
	return cmd
}

func newTopicCreateCommand(opts *rootOptions) *cobra.Command {
	topicOpts := &topicOptions{}
	var id, name, orientation string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a topic surface",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("topic id is required")
			}
			if name == "" {
				return errors.New("topic name is required")
			}
			topic := model.Topic{
				ID:          id,
				Name:        name,
				Orientation: orientation,
				CreatedAt:   nowUTC(),
				UpdatedAt:   nowUTC(),
			}
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			if err := store.SaveTopic(topic); err != nil {
				return err
			}
			return output(cmd, topicOpts.json, topic, formatTopic(topic))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "topic id")
	cmd.Flags().StringVar(&name, "name", "", "topic name")
	cmd.Flags().StringVar(&orientation, "orientation", "", "topic orientation")
	cmd.Flags().BoolVar(&topicOpts.json, "json", false, "output JSON")
	return cmd
}

func newTopicShowCommand(opts *rootOptions) *cobra.Command {
	topicOpts := &topicOptions{}
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			topic, err := store.LoadTopic(args[0])
			if err != nil {
				return err
			}
			return output(cmd, topicOpts.json, topic, formatTopic(topic))
		},
	}
	cmd.Flags().BoolVar(&topicOpts.json, "json", false, "output JSON")
	return cmd
}

func newTopicListCommand(opts *rootOptions) *cobra.Command {
	topicOpts := &topicOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List topics",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			topics, err := store.ListTopics()
			if err != nil {
				return err
			}
			items := make([]topicListItem, 0, len(topics))
			for _, topic := range topics {
				items = append(items, topicListItem{
					ID:          topic.ID,
					Name:        topic.Name,
					Orientation: topic.Orientation,
					UpdatedAt:   topic.UpdatedAt,
				})
			}
			if topicOpts.json {
				return output(cmd, true, items, "")
			}
			if len(items) == 0 {
				return output(cmd, false, nil, "no topics\n")
			}
			text := ""
			for _, item := range items {
				text += formatTopicListItem(item) + "\n"
			}
			return output(cmd, false, nil, text)
		},
	}
	cmd.Flags().BoolVar(&topicOpts.json, "json", false, "output JSON")
	return cmd
}

func newTopicOrientCommand(opts *rootOptions) *cobra.Command {
	topicOpts := &topicOptions{}
	var orientation string
	cmd := &cobra.Command{
		Use:   "orient <id>",
		Short: "Update a topic orientation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if orientation == "" {
				return errors.New("orientation is required")
			}
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			topic, err := store.LoadTopic(args[0])
			if err != nil {
				return err
			}
			topic.Orientation = orientation
			topic.UpdatedAt = nowUTC()
			if err := store.SaveTopic(topic); err != nil {
				return err
			}
			return output(cmd, topicOpts.json, topic, formatTopic(topic))
		},
	}
	cmd.Flags().StringVar(&orientation, "orientation", "", "topic orientation")
	cmd.Flags().BoolVar(&topicOpts.json, "json", false, "output JSON")
	return cmd
}

func newTopicPromoteCommand(opts *rootOptions) *cobra.Command {
	topicOpts := &topicOptions{}
	var threadID string
	var entryIDs []string
	var orientation string
	var includeSupporting bool

	cmd := &cobra.Command{
		Use:   "promote <id>",
		Short: "Promote durable understanding into a topic surface",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if threadID == "" && len(entryIDs) == 0 {
				return errors.New("promotion requires --thread or at least one --entry")
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			topic, err := store.LoadTopic(args[0])
			if err != nil {
				return err
			}

			if threadID != "" {
				thread, err := store.LoadThread(threadID)
				if err != nil {
					return err
				}
				if !contains(topic.ThreadIDs, thread.ID) {
					topic.ThreadIDs = append(topic.ThreadIDs, thread.ID)
				}
				if thread.ReentryAnchor != "" && !contains(topic.EntryIDs, thread.ReentryAnchor) {
					if _, err := store.LoadEntry(thread.ReentryAnchor); err == nil {
						topic.EntryIDs = append(topic.EntryIDs, thread.ReentryAnchor)
					}
				}
				if includeSupporting {
					for _, entryID := range thread.SupportingIDs {
						if _, err := store.LoadEntry(entryID); err != nil {
							return err
						}
						if !contains(topic.EntryIDs, entryID) {
							topic.EntryIDs = append(topic.EntryIDs, entryID)
						}
					}
				}
			}

			for _, entryID := range entryIDs {
				if _, err := store.LoadEntry(entryID); err != nil {
					return err
				}
				if !contains(topic.EntryIDs, entryID) {
					topic.EntryIDs = append(topic.EntryIDs, entryID)
				}
			}

			if orientation != "" {
				topic.Orientation = orientation
			}
			topic.UpdatedAt = nowUTC()

			if err := store.SaveTopic(topic); err != nil {
				return err
			}
			return output(cmd, topicOpts.json, topic, formatTopic(topic))
		},
	}
	cmd.Flags().StringVar(&threadID, "thread", "", "thread to promote from")
	cmd.Flags().StringSliceVar(&entryIDs, "entry", nil, "entries to attach to the topic")
	cmd.Flags().StringVar(&orientation, "orientation", "", "update topic orientation during promotion")
	cmd.Flags().BoolVar(&includeSupporting, "include-supporting", false, "include the thread's supporting entries")
	cmd.Flags().BoolVar(&topicOpts.json, "json", false, "output JSON")
	return cmd
}
