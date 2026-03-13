package cli

import (
	"fmt"

	"github.com/sandbox0-ai/sandnote/internal/index"
	"github.com/spf13/cobra"
)

func newIndexCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Work with the rebuildable derived index",
	}
	cmd.AddCommand(newIndexRebuildCommand(opts))
	return cmd
}

func newIndexRebuildCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the derived index from filesystem-backed truth",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			derived, err := index.Build(store)
			if err != nil {
				return err
			}
			if err := store.SaveDerivedIndex(derived); err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"rebuilt index at %s with %d threads, %d workspaces, %d topics\n",
				store.Root(),
				len(derived.Threads),
				len(derived.Workspaces),
				len(derived.Topics),
			)
			return nil
		},
	}
}
