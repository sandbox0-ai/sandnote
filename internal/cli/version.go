package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

type versionView struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
}

func newVersionCommand() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and build information for this preview",
		Example: joinLines(
			"  sandnote version",
			"  sandnote version --json",
			"  go build -ldflags \"-X github.com/sandbox0-ai/sandnote/internal/cli.Version=v0.1.0-preview -X github.com/sandbox0-ai/sandnote/internal/cli.GitCommit=$(git rev-parse --short HEAD) -X github.com/sandbox0-ai/sandnote/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)\" ./cmd/sandnote",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			view := versionView{
				Version:   Version,
				GitCommit: GitCommit,
				BuildDate: BuildDate,
			}
			if asJSON {
				data, err := json.MarshalIndent(view, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sandnote %s\n", view.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", view.GitCommit)
			fmt.Fprintf(cmd.OutOrStdout(), "build date: %s\n", view.BuildDate)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
