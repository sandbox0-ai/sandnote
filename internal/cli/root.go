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
		Short:         "CLI-first notebook engine for long-running agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: joinLines(
			"  sandnote version",
			"  sandnote init",
			"  sandnote overview",
			"  sandnote artifact import ./spec.md --id art_spec --mode reference",
			"  sandnote workspace create --id ws_auth --name task/auth",
			"  sandnote entry create --id en_auth --subject \"auth anchor\" --meaning \"resume auth work here\"",
			"  sandnote thread create --id th_auth --question \"How should auth work continue?\" --workspace ws_auth",
			"  sandnote entry attach en_auth --thread th_auth",
			"  sandnote entry link en_auth art_spec",
			"  sandnote workspace focus ws_auth th_auth",
			"  sandnote resume",
			"  sandnote repl",
		),
	}

	cmd.PersistentFlags().StringVar(
		&opts.storeRoot,
		"root",
		"",
		"filesystem root for sandnote state; defaults to the nearest initialized .sandnote",
	)

	cmd.AddCommand(
		newInitCommand(opts),
		newIndexCommand(opts),
		newVersionCommand(),
		newOverviewCommand(opts),
		newResumeCommand(opts),
		newArtifactCommand(opts),
		newEntryCommand(opts),
		newThreadCommand(opts),
		newWorkspaceCommand(opts),
		newTopicCommand(opts),
		newREPLCommand(opts),
	)

	return cmd
}

func resolvePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func currentWorkingDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(wd), nil
}

func resolveCommandStoreRoot(root string) (string, error) {
	if root != "" {
		return resolvePath(root)
	}

	wd, err := currentWorkingDirectory()
	if err != nil {
		return "", err
	}
	discovered, err := fsstore.DiscoverRoot(wd)
	if err != nil {
		return "", err
	}
	if discovered != "" {
		return discovered, nil
	}
	return filepath.Join(wd, ".sandnote"), nil
}

func resolveInitRootPath(rootPath, storeRoot string) (string, error) {
	if rootPath != "" {
		return resolvePath(rootPath)
	}
	if storeRoot != "" {
		resolvedStoreRoot, err := resolvePath(storeRoot)
		if err != nil {
			return "", err
		}
		return filepath.Dir(resolvedStoreRoot), nil
	}
	return currentWorkingDirectory()
}

func resolveInitStoreRoot(storeRoot, rootPath string) (string, error) {
	if storeRoot != "" {
		return resolvePath(storeRoot)
	}
	return filepath.Join(rootPath, ".sandnote"), nil
}

func newInitCommand(opts *rootOptions) *cobra.Command {
	var rootPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a filesystem-backed sandnote store",
		Example: joinLines(
			"  sandnote init",
			"  sandnote init --root-path /path/to/repo",
			"  sandnote --root /tmp/demo/.sandnote init",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedRootPath, err := resolveInitRootPath(rootPath, opts.storeRoot)
			if err != nil {
				return err
			}
			resolvedStoreRoot, err := resolveInitStoreRoot(opts.storeRoot, resolvedRootPath)
			if err != nil {
				return err
			}

			existingRoot, err := fsstore.DiscoverRoot(resolvedRootPath)
			if err != nil {
				return err
			}
			if existingRoot != "" && existingRoot != resolvedStoreRoot {
				return fmt.Errorf("sandnote store is already initialized at %s", existingRoot)
			}

			store := fsstore.New(resolvedStoreRoot)
			if err := store.Init(resolvedRootPath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized sandnote store at %s for root path %s\n", store.Root(), resolvedRootPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&rootPath, "root-path", "", "root path for notebook-relative discovery and artifact resolution")
	return cmd
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
