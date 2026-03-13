package fsstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

func TestInitCreatesLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))

	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if !store.Initialized() {
		t.Fatal("store should be initialized")
	}

	for _, path := range []string{
		filepath.Join(store.Root(), "entries"),
		filepath.Join(store.Root(), "threads"),
		filepath.Join(store.Root(), "workspaces"),
		filepath.Join(store.Root(), "topics"),
		filepath.Join(store.Root(), markerFile),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestSaveLoadEntryRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	entry := model.Entry{
		ID:             "en_123",
		Subject:        "checkpoint idea",
		State:          "draft",
		Meaning:        "Checkpoint should preserve continuity, not polish.",
		RelatedContext: []string{"th_123"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := store.SaveEntry(entry); err != nil {
		t.Fatalf("SaveEntry() error = %v", err)
	}

	got, err := store.LoadEntry(entry.ID)
	if err != nil {
		t.Fatalf("LoadEntry() error = %v", err)
	}

	if got.ID != entry.ID || got.Subject != entry.Subject || got.Meaning != entry.Meaning {
		t.Fatalf("loaded entry mismatch: got %+v want %+v", got, entry)
	}
}

func TestSaveLoadThreadRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	thread := model.Thread{
		ID:            "th_123",
		Question:      "Why is this thread still live?",
		CurrentBelief: "Checkpoint quality is the v0 center.",
		OpenEdge:      "Need to define command boundaries.",
		NextLean:      "Clarify resume vs inspect.",
		ReentryAnchor: "comment:thread-loop",
		Vitality:      model.VitalityLive,
		WorkspaceID:   "ws_123",
		SupportingIDs: []string{"en_1", "en_2"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := store.SaveThread(thread); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	got, err := store.LoadThread(thread.ID)
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}

	if got.ID != thread.ID || got.Question != thread.Question || got.Vitality != thread.Vitality {
		t.Fatalf("loaded thread mismatch: got %+v want %+v", got, thread)
	}
	if got.ReentryAnchor != thread.ReentryAnchor || got.NextLean != thread.NextLean {
		t.Fatalf("loaded thread continuity mismatch: got %+v want %+v", got, thread)
	}
}

func TestSaveLoadWorkspaceRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	workspace := model.Workspace{
		ID:            "ws_123",
		Name:          "task/auth-investigation",
		FocusThreadID: "th_123",
		ThreadIDs:     []string{"th_123", "th_456"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := store.SaveWorkspace(workspace); err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}

	got, err := store.LoadWorkspace(workspace.ID)
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}

	if got.ID != workspace.ID || got.Name != workspace.Name || got.FocusThreadID != workspace.FocusThreadID {
		t.Fatalf("loaded workspace mismatch: got %+v want %+v", got, workspace)
	}
}

func TestSaveLoadTopicRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	topic := model.Topic{
		ID:          "tp_auth",
		Name:        "auth-boundaries",
		Orientation: "Start here when a task touches auth and permissions.",
		EntryIDs:    []string{"en_123"},
		ThreadIDs:   []string{"th_123"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.SaveTopic(topic); err != nil {
		t.Fatalf("SaveTopic() error = %v", err)
	}

	got, err := store.LoadTopic(topic.ID)
	if err != nil {
		t.Fatalf("LoadTopic() error = %v", err)
	}

	if got.ID != topic.ID || got.Name != topic.Name || got.Orientation != topic.Orientation {
		t.Fatalf("loaded topic mismatch: got %+v want %+v", got, topic)
	}
}

func TestSaveThreadRejectsInvalidVitality(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	thread := model.Thread{
		ID:       "th_invalid",
		Question: "bad vitality",
		Vitality: "broken",
	}

	if err := store.SaveThread(thread); err == nil {
		t.Fatal("SaveThread() expected validation error")
	}
}

func TestSaveBeforeInitFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))

	entry := model.Entry{
		ID:        "en_1",
		Subject:   "seed note",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := store.SaveEntry(entry); err == nil {
		t.Fatal("SaveEntry() expected initialization error")
	}
}
