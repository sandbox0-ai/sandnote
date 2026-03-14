package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func updateActiveWorkSelection(store *fsstore.Store, workspaceID, threadID string) error {
	state, err := loadREPLState(store)
	if err != nil {
		return err
	}
	state.currentWorkspace = workspaceID
	state.focusThread = threadID
	return saveREPLState(store, state)
}

func alignActiveSelectionAfterTransition(store *fsstore.Store, thread model.Thread) error {
	if thread.Vitality == model.VitalityLive {
		return nil
	}

	nextFocus := ""
	if strings.TrimSpace(thread.WorkspaceID) != "" {
		workspace, err := store.LoadWorkspace(thread.WorkspaceID)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else {
			if workspace.FocusThreadID == thread.ID {
				items, err := buildFrontier(store, thread.WorkspaceID)
				if err != nil {
					return err
				}
				if len(items) > 0 {
					nextFocus = items[0].ID
				}
				workspace.FocusThreadID = nextFocus
				workspace.UpdatedAt = nowUTC()
				if err := store.SaveWorkspace(workspace); err != nil {
					return err
				}
			} else {
				nextFocus = workspace.FocusThreadID
			}
		}
	}

	state, err := loadREPLState(store)
	if err != nil {
		return err
	}
	if state.focusThread != thread.ID {
		return nil
	}
	if state.currentWorkspace == thread.WorkspaceID {
		state.focusThread = nextFocus
	} else {
		state.focusThread = ""
	}
	state.inspectionScope = nil
	state.pendingCheckpointContext = ""
	return saveREPLState(store, state)
}

func alignActiveSelectionAfterWorkspaceDetach(store *fsstore.Store, workspace model.Workspace, detachedThreadID string) error {
	nextFocus := workspace.FocusThreadID

	state, err := loadREPLState(store)
	if err != nil {
		return err
	}
	if state.focusThread != detachedThreadID {
		return nil
	}

	if state.currentWorkspace == workspace.ID {
		state.focusThread = nextFocus
	} else {
		state.focusThread = ""
	}
	state.inspectionScope = nil
	state.pendingCheckpointContext = ""
	return saveREPLState(store, state)
}
