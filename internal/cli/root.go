package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	storeRoot string
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:           "sandnote",
		Short:         "CLI-first notebook engine for agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: joinLines(
			"  sandnote init",
			"  sandnote workspace create --id ws_auth --name task/auth",
			"  sandnote entry create --id en_auth --subject \"auth anchor\" --meaning \"resume auth work here\"",
			"  sandnote thread create --id th_auth --question \"How should auth work continue?\" --workspace ws_auth",
			"  sandnote entry attach en_auth --thread th_auth",
			"  sandnote workspace focus ws_auth th_auth",
			"  sandnote resume",
			"  sandnote repl",
		),
	}

	cmd.PersistentFlags().StringVar(
		&opts.storeRoot,
		"root",
		defaultStoreRoot(),
		"filesystem root for sandnote state",
	)

	cmd.AddCommand(
		newInitCommand(opts),
		newIndexCommand(opts),
		newResumeCommand(opts),
		newEntryCommand(opts),
		newThreadCommand(opts),
		newWorkspaceCommand(opts),
		newTopicCommand(opts),
		newREPLCommand(opts),
	)

	return cmd
}

func defaultStoreRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".sandnote"
	}
	return filepath.Join(wd, ".sandnote")
}

func newInitCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a filesystem-backed sandnote store",
		Example: joinLines(
			"  sandnote init",
			"  sandnote --root /tmp/demo/.sandnote init",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := fsstore.New(opts.storeRoot)
			if err := store.Init(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized sandnote store at %s\n", store.Root())
			return nil
		},
	}
}

func addNotImplementedSubcommands(parent *cobra.Command, names ...string) {
	for _, name := range names {
		parent.AddCommand(&cobra.Command{
			Use:   name,
			Short: fmt.Sprintf("%s %s", parent.Use, name),
			RunE:  notImplemented(parent.Use + " " + name),
		})
	}
}

func notImplemented(name string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("%s is not implemented yet", name)
	}
}
