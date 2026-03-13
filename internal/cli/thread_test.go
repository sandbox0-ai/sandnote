package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func TestThreadShowJSON(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root, "thread", "show", "th_123", "--json")

	var got threadShowView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.ID != "th_123" || got.Vitality != model.VitalityLive {
		t.Fatalf("unexpected show output: %+v", got)
	}
}

func TestThreadResumeText(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root, "thread", "resume", "th_123")
	text := output.String()

	for _, want := range []string{
		"resume th_123",
		"current belief: Checkpoint quality is the center.",
		"open edge: Need to separate resume from inspect.",
		"next lean: Define command boundaries.",
		"re-entry anchor: en_anchor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("resume output missing %q:\n%s", want, text)
		}
	}
}

func TestThreadInspectJSONIncludesSupportingEntries(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root, "thread", "inspect", "th_123", "--json")

	var got threadInspectView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.SupportingEntries) != 2 {
		t.Fatalf("expected 2 supporting entries, got %d", len(got.SupportingEntries))
	}
}

func TestThreadCheckpointUpdatesContinuityState(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root,
		"thread", "checkpoint", "th_123",
		"--belief", "The CLI shape is stable.",
		"--open-edge", "Need to implement transition tests.",
		"--next-lean", "Add JSON output coverage.",
		"--reentry-anchor", "en_support_2",
		"--json",
	)

	var got threadResumeView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.CurrentBelief != "The CLI shape is stable." || got.ReentryAnchor != "en_support_2" {
		t.Fatalf("unexpected checkpoint output: %+v", got)
	}

	store := fsstore.New(root)
	thread, err := store.LoadThread("th_123")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.NextLean != "Add JSON output coverage." || thread.OpenEdge != "Need to implement transition tests." {
		t.Fatalf("thread not updated: %+v", thread)
	}
}

func TestThreadTransitionUpdatesVitality(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root, "thread", "transition", "th_123", "--to", "dormant", "--json")

	var got threadShowView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Vitality != model.VitalityDormant {
		t.Fatalf("unexpected vitality: %+v", got)
	}
}

func TestThreadListFiltersByVitality(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	output := executeCLI(t, root, "thread", "list", "--vitality", "live", "--json")

	var got []threadListItem
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "th_123" {
		t.Fatalf("unexpected list output: %+v", got)
	}
}

func TestThreadCheckpointRequiresContinuityFlags(t *testing.T) {
	t.Parallel()

	root := seedThreadStore(t)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--root", root, "thread", "checkpoint", "th_123"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "checkpoint requires at least one") {
		t.Fatalf("expected checkpoint validation error, got %v", err)
	}
}

func executeCLI(t *testing.T, root string, args ...string) *bytes.Buffer {
	t.Helper()

	cmd := NewRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs(append([]string{"--root", root}, args...))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, output.String())
	}
	return output
}

func seedThreadStore(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), ".sandnote")
	store := fsstore.New(root)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	entries := []model.Entry{
		{
			ID:        "en_anchor",
			Subject:   "thread loop anchor",
			State:     "draft",
			Meaning:   "Resume, inspect, checkpoint, and transition need separate semantics.",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "en_support_2",
			Subject:   "checkpoint semantics",
			State:     "draft",
			Meaning:   "A checkpoint should leave a better edge.",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	for _, entry := range entries {
		if err := store.SaveEntry(entry); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
	}

	threads := []model.Thread{
		{
			ID:            "th_123",
			Question:      "How should the canonical thread loop behave?",
			CurrentBelief: "Checkpoint quality is the center.",
			OpenEdge:      "Need to separate resume from inspect.",
			NextLean:      "Define command boundaries.",
			ReentryAnchor: "en_anchor",
			Vitality:      model.VitalityLive,
			WorkspaceID:   "ws_cli",
			SupportingIDs: []string{"en_anchor", "en_support_2"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:        "th_456",
			Question:  "What should be deferred to later issues?",
			Vitality:  model.VitalityDormant,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	for _, thread := range threads {
		if err := store.SaveThread(thread); err != nil {
			t.Fatalf("SaveThread() error = %v", err)
		}
	}

	return root
}
