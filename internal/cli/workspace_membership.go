package cli

import (
	"errors"
	"os"
	"slices"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func derivedWorkspaceThreadIDs(store *fsstore.Store, workspaceID string) ([]string, error) {
	threads, err := store.ListThreads()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	for _, thread := range threads {
		if thread.WorkspaceID == workspaceID {
			ids = append(ids, thread.ID)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

func withDerivedWorkspaceMembership(store *fsstore.Store, workspace model.Workspace) (model.Workspace, error) {
	threadIDs, err := derivedWorkspaceThreadIDs(store, workspace.ID)
	if err != nil {
		return model.Workspace{}, err
	}
	workspace.ThreadIDs = threadIDs
	if workspace.FocusThreadID != "" && !contains(threadIDs, workspace.FocusThreadID) {
		workspace.FocusThreadID = ""
	}
	return workspace, nil
}

func syncWorkspaceMembership(store *fsstore.Store, workspaceID string) (model.Workspace, error) {
	workspace, err := store.LoadWorkspace(workspaceID)
	if err != nil {
		return model.Workspace{}, err
	}

	derived, err := withDerivedWorkspaceMembership(store, workspace)
	if err != nil {
		return model.Workspace{}, err
	}
	if workspace.FocusThreadID == derived.FocusThreadID && slices.Equal(workspace.ThreadIDs, derived.ThreadIDs) {
		return workspace, nil
	}

	derived.UpdatedAt = nowUTC()
	if err := store.SaveWorkspace(derived); err != nil {
		return model.Workspace{}, err
	}
	return derived, nil
}

func syncWorkspaceMembershipIfExists(store *fsstore.Store, workspaceID string) error {
	if workspaceID == "" {
		return nil
	}
	_, err := syncWorkspaceMembership(store, workspaceID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
