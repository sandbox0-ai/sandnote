package cli

import (
	"errors"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/spf13/cobra"
)

type workspaceOptions struct {
	json  bool
	query string
}

func newWorkspaceCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage current thinking context",
	}
	cmd.AddCommand(
		newWorkspaceCreateCommand(opts),
		newWorkspaceShowCommand(opts),
		newWorkspaceListCommand(opts),
		newWorkspaceUseCommand(opts),
		newWorkspaceThreadsCommand(opts),
		newWorkspaceAttachCommand(opts),
		newWorkspaceDetachCommand(opts),
		newWorkspaceFocusCommand(opts),
	)
	return cmd
}

func newWorkspaceCreateCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	var id, name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("workspace id is required")
			}
			if name == "" {
				return errors.New("workspace name is required")
			}
			workspace := model.Workspace{
				ID:        id,
				Name:      name,
				CreatedAt: nowUTC(),
				UpdatedAt: nowUTC(),
			}
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			if err := store.SaveWorkspace(workspace); err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "workspace id")
	cmd.Flags().StringVar(&name, "name", "", "workspace name")
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceShowCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			workspace, err = withDerivedWorkspaceMembership(store, workspace)
			if err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceListCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			derived, err := loadOrBuildIndex(store)
			if err != nil {
				return err
			}
			items := make([]workspaceListItem, 0, len(derived.Workspaces))
			for _, workspace := range derived.Workspaces {
				if !matchesQuery(workspaceOpts.query, workspace.ID, workspace.Name, workspace.FocusThreadID) {
					continue
				}
				items = append(items, workspaceListItem{
					ID:            workspace.ID,
					Name:          workspace.Name,
					FocusThreadID: workspace.FocusThreadID,
					ThreadCount:   workspace.ThreadCount,
					UpdatedAt:     workspace.UpdatedAt,
				})
			}
			if workspaceOpts.json {
				return output(cmd, true, items, "")
			}
			if len(items) == 0 {
				return output(cmd, false, nil, "no workspaces\n")
			}
			text := ""
			for _, item := range items {
				text += formatWorkspaceListItem(item) + "\n"
			}
			return output(cmd, false, nil, text)
		},
	}
	cmd.Flags().StringVar(&workspaceOpts.query, "query", "", "filter by text query")
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceUseCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "use <id>",
		Short: "Set the current workspace for canonical CLI and REPL flows",
		Example: joinLines(
			"  sandnote workspace use ws_auth",
			"  sandnote workspace use ws_auth --json",
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			workspace, err = withDerivedWorkspaceMembership(store, workspace)
			if err != nil {
				return err
			}
			if err := updateActiveWorkSelection(store, workspace.ID, workspace.FocusThreadID); err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceThreadsCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "threads <id>",
		Short: "List threads relevant to a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			threads, err := store.ListThreads()
			if err != nil {
				return err
			}

			items := make([]threadListItem, 0)
			for _, thread := range threads {
				if thread.WorkspaceID != workspace.ID {
					continue
				}
				items = append(items, threadListItem{
					ID:            thread.ID,
					Question:      thread.Question,
					Vitality:      thread.Vitality,
					WorkspaceID:   thread.WorkspaceID,
					UpdatedAt:     thread.UpdatedAt,
					CurrentBelief: thread.CurrentBelief,
					OpenEdge:      thread.OpenEdge,
				})
			}

			if workspaceOpts.json {
				return output(cmd, true, items, "")
			}
			if len(items) == 0 {
				return output(cmd, false, nil, "no threads\n")
			}
			text := ""
			for _, item := range items {
				text += formatThreadListItem(item) + "\n"
			}
			return output(cmd, false, nil, text)
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceAttachCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "attach <workspace-id> <thread-id>",
		Short: "Attach a thread to a workspace without changing focus",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[1])
			if err != nil {
				return err
			}
			if thread.WorkspaceID != "" && thread.WorkspaceID != workspace.ID {
				return errors.New("thread belongs to a different workspace")
			}

			now := nowUTC()
			threadChanged := false
			if thread.WorkspaceID != workspace.ID {
				thread.WorkspaceID = workspace.ID
				thread.UpdatedAt = now
				threadChanged = true
			}
			if threadChanged {
				if err := store.SaveThread(thread); err != nil {
					return err
				}
			}
			workspace, err = syncWorkspaceMembership(store, workspace.ID)
			if err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceDetachCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "detach <workspace-id> <thread-id>",
		Short: "Detach a thread from a workspace without deleting either object",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[1])
			if err != nil {
				return err
			}

			attached := thread.WorkspaceID == workspace.ID
			if !attached {
				return errors.New("thread is not attached to this workspace")
			}

			now := nowUTC()
			if thread.WorkspaceID == workspace.ID {
				thread.WorkspaceID = ""
				thread.UpdatedAt = now
				if err := store.SaveThread(thread); err != nil {
					return err
				}
			}

			focusCleared := workspace.FocusThreadID == thread.ID
			if focusCleared {
				workspace.FocusThreadID = ""
			}
			workspace.UpdatedAt = now
			if err := store.SaveWorkspace(workspace); err != nil {
				return err
			}
			workspace, err = syncWorkspaceMembership(store, workspace.ID)
			if err != nil {
				return err
			}

			if focusCleared {
				items, err := buildFrontier(store, workspace.ID)
				if err != nil {
					return err
				}
				if len(items) > 0 {
					workspace.FocusThreadID = items[0].ID
					workspace.UpdatedAt = nowUTC()
					if err := store.SaveWorkspace(workspace); err != nil {
						return err
					}
				}
			}

			if err := alignActiveSelectionAfterWorkspaceDetach(store, workspace, thread.ID); err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}

func newWorkspaceFocusCommand(opts *rootOptions) *cobra.Command {
	workspaceOpts := &workspaceOptions{}
	cmd := &cobra.Command{
		Use:   "focus <workspace-id> <thread-id>",
		Short: "Set the focus thread for a workspace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			workspace, err := store.LoadWorkspace(args[0])
			if err != nil {
				return err
			}
			thread, err := store.LoadThread(args[1])
			if err != nil {
				return err
			}
			if thread.WorkspaceID != "" && thread.WorkspaceID != workspace.ID {
				return errors.New("thread belongs to a different workspace")
			}
			if thread.WorkspaceID == "" {
				thread.WorkspaceID = workspace.ID
				thread.UpdatedAt = nowUTC()
				if err := store.SaveThread(thread); err != nil {
					return err
				}
			}
			workspace.FocusThreadID = thread.ID
			workspace.UpdatedAt = nowUTC()
			if err := store.SaveWorkspace(workspace); err != nil {
				return err
			}
			workspace, err = syncWorkspaceMembership(store, workspace.ID)
			if err != nil {
				return err
			}
			return output(cmd, workspaceOpts.json, workspace, formatWorkspace(workspace))
		},
	}
	cmd.Flags().BoolVar(&workspaceOpts.json, "json", false, "output JSON")
	return cmd
}
