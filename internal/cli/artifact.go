package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/spf13/cobra"
)

type artifactOptions struct {
	json  bool
	query string
}

func newArtifactCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Manage source-backed content without overloading entries",
	}
	cmd.AddCommand(
		newArtifactImportCommand(opts),
		newArtifactShowCommand(opts),
		newArtifactListCommand(opts),
	)
	return cmd
}

func newArtifactImportCommand(opts *rootOptions) *cobra.Command {
	artifactOpts := &artifactOptions{}
	var id, kind, mode string
	var entryIDs []string

	cmd := &cobra.Command{
		Use:   "import <path>",
		Short: "Import a local document as an artifact reference or snapshot",
		Example: joinLines(
			"  sandnote artifact import ./diagd-spec.md --id art_diagd --mode reference",
			"  sandnote artifact import ./diagd-spec.md --id art_diagd_snapshot --mode snapshot",
			"  sandnote artifact import ./diagd-spec.md --id art_diagd --entry en_auth",
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return errors.New("artifact id is required")
			}

			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}

			entries := make([]model.Entry, 0, len(entryIDs))
			for _, entryID := range entryIDs {
				entry, err := store.LoadEntry(entryID)
				if err != nil {
					return err
				}
				entries = append(entries, entry)
			}

			sourceRef, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			data, err := os.ReadFile(sourceRef)
			if err != nil {
				return err
			}
			info, err := os.Stat(sourceRef)
			if err != nil {
				return err
			}

			artifactKind := strings.TrimSpace(kind)
			if artifactKind == "" {
				artifactKind = inferArtifactKind(sourceRef)
			}
			if artifactKind == "" {
				artifactKind = "other"
			}

			ingestMode := model.ArtifactIngestMode(mode)
			if strings.TrimSpace(mode) == "" {
				ingestMode = model.ArtifactReference
			}

			artifact := model.Artifact{
				ID:         id,
				Kind:       artifactKind,
				SourceRef:  sourceRef,
				IngestMode: ingestMode,
				CreatedAt:  nowUTC(),
				UpdatedAt:  nowUTC(),
			}
			prepareArtifactReference(opts.storeRoot, &artifact, data, info)
			if artifact.IngestMode == model.ArtifactSnapshot {
				artifact.Body = string(data)
			}

			if err := store.SaveArtifact(artifact); err != nil {
				return err
			}

			for _, entry := range entries {
				if contains(entry.RelatedContext, artifact.ID) {
					continue
				}
				entry.RelatedContext = append(entry.RelatedContext, artifact.ID)
				entry.UpdatedAt = nowUTC()
				if err := store.SaveEntry(entry); err != nil {
					return err
				}
			}

			return output(cmd, artifactOpts.json, artifact, formatArtifact(artifact))
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "artifact id")
	cmd.Flags().StringVar(&kind, "kind", "", "artifact kind override")
	cmd.Flags().StringVar(&mode, "mode", string(model.ArtifactReference), "artifact ingest mode: reference or snapshot")
	cmd.Flags().StringSliceVar(&entryIDs, "entry", nil, "attach imported artifact to one or more entries")
	cmd.Flags().BoolVar(&artifactOpts.json, "json", false, "output JSON")
	return cmd
}

func newArtifactShowCommand(opts *rootOptions) *cobra.Command {
	artifactOpts := &artifactOptions{}
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show an artifact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			artifact, err := store.LoadArtifact(args[0])
			if err != nil {
				return err
			}
			artifact, changed, err := resolveArtifactReference(opts.storeRoot, artifact)
			if err != nil {
				return err
			}
			if changed {
				if err := store.SaveArtifact(artifact); err != nil {
					return err
				}
			}
			return output(cmd, artifactOpts.json, artifact, formatArtifact(artifact))
		},
	}
	cmd.Flags().BoolVar(&artifactOpts.json, "json", false, "output JSON")
	return cmd
}

func newArtifactListCommand(opts *rootOptions) *cobra.Command {
	artifactOpts := &artifactOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			artifacts, err := store.ListArtifacts()
			if err != nil {
				return err
			}

			items := make([]artifactListItem, 0, len(artifacts))
			for _, artifact := range artifacts {
				resolved, changed, err := resolveArtifactReference(opts.storeRoot, artifact)
				if err != nil {
					return err
				}
				if changed {
					if err := store.SaveArtifact(resolved); err != nil {
						return err
					}
				}
				if !matchesQuery(artifactOpts.query, resolved.ID, resolved.Kind, resolved.SourceRef, string(resolved.IngestMode)) {
					continue
				}
				items = append(items, artifactListItem{
					ID:         resolved.ID,
					Kind:       resolved.Kind,
					SourceRef:  resolved.SourceRef,
					IngestMode: resolved.IngestMode,
					UpdatedAt:  resolved.UpdatedAt,
				})
			}

			if artifactOpts.json {
				return output(cmd, true, items, "")
			}
			if len(items) == 0 {
				return output(cmd, false, nil, "no artifacts\n")
			}

			text := ""
			for _, item := range items {
				text += formatArtifactListItem(item) + "\n"
			}
			return output(cmd, false, nil, text)
		},
	}
	cmd.Flags().StringVar(&artifactOpts.query, "query", "", "filter by text query")
	cmd.Flags().BoolVar(&artifactOpts.json, "json", false, "output JSON")
	return cmd
}

func inferArtifactKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdx":
		return "markdown"
	case ".txt":
		return "text"
	case ".json":
		return "json"
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".py", ".rb", ".rs", ".java", ".c", ".cc", ".cpp", ".h", ".hpp", ".sh", ".bash", ".zsh", ".sql", ".yaml", ".yml", ".toml":
		return "code"
	default:
		return "other"
	}
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
