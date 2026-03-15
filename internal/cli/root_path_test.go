package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func executeCLIInDir(t *testing.T, dir string, args ...string) (*bytes.Buffer, error) {
	t.Helper()

	t.Chdir(dir)

	cmd := NewRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return output, err
}

func TestInitPersistsRootPathAndRejectsNestedRepeat(t *testing.T) {
	workspace := t.TempDir()
	nested := filepath.Join(workspace, "services", "manager")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	output, err := executeCLIInDir(t, workspace, "init")
	if err != nil {
		t.Fatalf("init error = %v\noutput=%s", err, output.String())
	}

	storeRoot := filepath.Join(workspace, ".sandnote")
	if !strings.Contains(output.String(), "initialized sandnote store at "+storeRoot+" for root path "+workspace) {
		t.Fatalf("unexpected init output:\n%s", output.String())
	}

	store := fsstore.New(storeRoot)
	marker, err := store.LoadMarker()
	if err != nil {
		t.Fatalf("LoadMarker() error = %v", err)
	}
	if marker.RootPath != workspace {
		t.Fatalf("expected root path %q, got %+v", workspace, marker)
	}

	_, err = executeCLIInDir(t, nested, "init")
	if err == nil || !strings.Contains(err.Error(), "sandnote store is already initialized at "+storeRoot) {
		t.Fatalf("expected nested repeat init to fail with existing store, got %v", err)
	}
}

func TestCommandsDiscoverStoreAndUsePersistedRootPath(t *testing.T) {
	workspace := t.TempDir()
	nested := filepath.Join(workspace, "apps", "dashboard")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(nested) error = %v", err)
	}

	if _, err := executeCLIInDir(t, workspace, "init"); err != nil {
		t.Fatalf("init error = %v", err)
	}

	sourceDir := filepath.Join(workspace, "docs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sourceDir) error = %v", err)
	}
	oldPath := filepath.Join(sourceDir, "diagd-spec.md")
	if err := os.WriteFile(oldPath, []byte("# diagd\ntrusted capability broker\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(oldPath) error = %v", err)
	}

	importPath := filepath.Join("..", "..", "docs", "diagd-spec.md")
	if _, err := executeCLIInDir(t, nested, "artifact", "import", importPath, "--id", "art_diagd", "--mode", "reference"); err != nil {
		t.Fatalf("artifact import error = %v", err)
	}

	archiveDir := filepath.Join(workspace, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(archiveDir) error = %v", err)
	}
	newPath := filepath.Join(archiveDir, "diagd-spec-renamed.md")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	output, err := executeCLIInDir(t, nested, "artifact", "show", "art_diagd", "--json")
	if err != nil {
		t.Fatalf("artifact show error = %v\noutput=%s", err, output.String())
	}

	var got model.Artifact
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.SourceRef != newPath {
		t.Fatalf("expected relocated artifact source_ref %q, got %+v", newPath, got)
	}
}
