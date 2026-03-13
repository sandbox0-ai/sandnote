package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type resumeOptions struct {
	json      bool
	workspace string
}

type resumeView struct {
	Thread               threadResumeView `json:"thread"`
	WorkspaceID          string           `json:"workspace_id,omitempty"`
	ContinuationPressure int              `json:"continuation_pressure"`
	Reasons              []string         `json:"reasons,omitempty"`
}

func newResumeCommand(opts *rootOptions) *cobra.Command {
	resumeOpts := &resumeOptions{}
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume the strongest live thread in the current work frontier",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}

			items, err := buildFrontier(store, resumeOpts.workspace)
			if err != nil {
				return err
			}
			best, err := bestFrontierItem(items)
			if err != nil {
				return err
			}

			thread, err := store.LoadThread(best.ID)
			if err != nil {
				return err
			}

			view := resumeView{
				Thread: threadResumeView{
					ID:            thread.ID,
					Question:      thread.Question,
					Vitality:      thread.Vitality,
					CurrentBelief: thread.CurrentBelief,
					OpenEdge:      thread.OpenEdge,
					NextLean:      thread.NextLean,
					ReentryAnchor: thread.ReentryAnchor,
				},
				WorkspaceID:          thread.WorkspaceID,
				ContinuationPressure: best.ContinuationPressure,
				Reasons:              best.Reasons,
			}

			text := formatThreadResume(thread)
			if thread.WorkspaceID != "" {
				text += fmt.Sprintf("workspace: %s\n", thread.WorkspaceID)
			}
			if best.ContinuationPressure > 0 {
				text += fmt.Sprintf("continuation pressure: %d\n", best.ContinuationPressure)
			}
			if len(best.Reasons) > 0 {
				text += fmt.Sprintf("reasons: %s\n", joinCSV(best.Reasons))
			}
			return output(cmd, resumeOpts.json, view, text)
		},
	}
	cmd.Flags().StringVar(&resumeOpts.workspace, "workspace", "", "prefer live threads from a workspace")
	cmd.Flags().BoolVar(&resumeOpts.json, "json", false, "output JSON")
	return cmd
}
