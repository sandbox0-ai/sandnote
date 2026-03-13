package cli

import "github.com/sandbox0-ai/sandnote/internal/store/fsstore"

func updateActiveWorkSelection(store *fsstore.Store, workspaceID, threadID string) error {
	state, err := loadREPLState(store)
	if err != nil {
		return err
	}
	if workspaceID != "" {
		state.currentWorkspace = workspaceID
	}
	state.focusThread = threadID
	return saveREPLState(store, state)
}
