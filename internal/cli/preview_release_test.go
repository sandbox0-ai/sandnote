package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

func TestVersionCommandExposesPreviewBuildMetadata(t *testing.T) {
	t.Parallel()

	originalVersion := Version
	originalCommit := GitCommit
	originalBuildDate := BuildDate
	Version = "v0.1.0-preview"
	GitCommit = "abc1234"
	BuildDate = "2026-03-14T18:00:00Z"
	defer func() {
		Version = originalVersion
		GitCommit = originalCommit
		BuildDate = originalBuildDate
	}()

	textOutput, err := executeCLIAllowError(filepath.Join(t.TempDir(), ".sandnote"), "version")
	if err != nil {
		t.Fatalf("version text command error = %v\noutput=%s", err, textOutput.String())
	}
	for _, want := range []string{
		"sandnote v0.1.0-preview",
		"commit: abc1234",
		"build date: 2026-03-14T18:00:00Z",
	} {
		if !strings.Contains(textOutput.String(), want) {
			t.Fatalf("version text output missing %q:\n%s", want, textOutput.String())
		}
	}

	jsonOutput, err := executeCLIAllowError(filepath.Join(t.TempDir(), ".sandnote"), "version", "--json")
	if err != nil {
		t.Fatalf("version json command error = %v\noutput=%s", err, jsonOutput.String())
	}
	var got versionView
	if err := json.Unmarshal(jsonOutput.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Version != "v0.1.0-preview" || got.GitCommit != "abc1234" || got.BuildDate != "2026-03-14T18:00:00Z" {
		t.Fatalf("unexpected version json output: %+v", got)
	}
}

func TestPreviewSmokeFlow(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), ".sandnote")
	executeCLI(t, root, "init")
	executeCLI(t, root, "workspace", "create", "--id", "ws_auth", "--name", "task/auth")
	executeCLI(t, root, "entry", "create", "--id", "en_auth", "--subject", "auth anchor", "--meaning", "resume auth work here")
	executeCLI(t, root, "thread", "create", "--id", "th_auth", "--question", "How should auth work continue?", "--workspace", "ws_auth")
	executeCLI(t, root, "entry", "attach", "en_auth", "--thread", "th_auth")
	executeCLI(t, root, "workspace", "focus", "ws_auth", "th_auth")
	executeCLI(t, root, "workspace", "use", "ws_auth")

	resumeOutput := executeCLI(t, root, "resume")
	if !strings.Contains(resumeOutput.String(), "resume th_auth") {
		t.Fatalf("expected resume output for th_auth:\n%s", resumeOutput.String())
	}

	executeCLI(
		t,
		root,
		"thread", "checkpoint", "th_auth",
		"--belief", "auth flow is working",
		"--open-edge", "promote durable auth understanding",
		"--next-lean", "promote auth topic",
		"--reentry-anchor", "en_auth",
	)

	executeCLI(t, root, "topic", "create", "--id", "tp_auth", "--name", "auth", "--orientation", "Start here for auth work.")
	executeCLI(t, root, "topic", "promote", "tp_auth", "--thread", "th_auth", "--include-supporting")
	indexOutput := executeCLI(t, root, "index", "rebuild")
	if !strings.Contains(indexOutput.String(), "rebuilt index at") {
		t.Fatalf("expected index rebuild confirmation:\n%s", indexOutput.String())
	}

	topicEntriesOutput := executeCLI(t, root, "topic", "entries", "tp_auth", "--json")
	var entries []model.Entry
	if err := json.Unmarshal(topicEntriesOutput.Bytes(), &entries); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "en_auth" {
		t.Fatalf("unexpected preview topic entries: %+v", entries)
	}
}

func TestRootHelpMentionsVersionAndPreviewWorkflow(t *testing.T) {
	t.Parallel()

	cmd := NewRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--help) error = %v\noutput=%s", err, output.String())
	}
	for _, want := range []string{
		"sandnote version",
		"sandnote init",
		"sandnote repl",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("root help missing %q:\n%s", want, output.String())
		}
	}
}
