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
	}

	cmd.PersistentFlags().StringVar(
		&opts.storeRoot,
		"root",
		defaultStoreRoot(),
		"filesystem root for sandnote state",
	)

	cmd.AddCommand(
		newInitCommand(opts),
		newEntryCommand(),
		newThreadCommand(opts),
		newWorkspaceCommand(),
		newTopicCommand(),
		newREPLCommand(),
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

func newEntryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entry",
		Short: "Manage lightweight thinking units",
	}
	addNotImplementedSubcommands(cmd,
		"create",
		"show",
		"list",
		"revise",
		"link",
		"attach",
		"archive",
	)
	return cmd
}

func newWorkspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage current thinking context",
	}
	addNotImplementedSubcommands(cmd,
		"create",
		"show",
		"list",
		"use",
		"threads",
		"focus",
		"attach",
		"detach",
	)
	return cmd
}

func newTopicCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Manage durable re-entry surfaces",
	}
	addNotImplementedSubcommands(cmd,
		"create",
		"show",
		"list",
		"orient",
		"promote",
		"entries",
		"threads",
	)
	return cmd
}

func newREPLCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Start the stateful working console",
		RunE:  notImplemented("repl"),
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
