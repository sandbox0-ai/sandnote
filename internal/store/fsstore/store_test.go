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
		filepath.Join(store.Root(), "artifacts"),
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

func TestSaveLoadArtifactRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC().Round(time.Second)
	artifact := model.Artifact{
		ID:            "art_123",
		Kind:          "markdown",
		SourceRef:     "/tmp/spec.md",
		IngestMode:    model.ArtifactSnapshot,
		ContentDigest: "sha256:abc123",
		Body:          "# spec\n",
		Locator: &model.ArtifactLocator{
			SearchRoots:     []string{"/tmp"},
			SizeBytes:       7,
			ModTimeUnixNano: now.UnixNano(),
			FileIdentity: &model.FileIdentity{
				Kind:     "posix_inode",
				DeviceID: 1,
				ObjectID: 2,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.SaveArtifact(artifact); err != nil {
		t.Fatalf("SaveArtifact() error = %v", err)
	}

	got, err := store.LoadArtifact(artifact.ID)
	if err != nil {
		t.Fatalf("LoadArtifact() error = %v", err)
	}

	if got.ID != artifact.ID || got.Kind != artifact.Kind || got.IngestMode != artifact.IngestMode {
		t.Fatalf("loaded artifact mismatch: got %+v want %+v", got, artifact)
	}
	if got.ContentDigest != artifact.ContentDigest || got.Body != artifact.Body {
		t.Fatalf("loaded artifact content mismatch: got %+v want %+v", got, artifact)
	}
	if got.Locator == nil || got.Locator.FileIdentity == nil || got.Locator.FileIdentity.ObjectID != 2 {
		t.Fatalf("loaded artifact locator mismatch: got %+v want %+v", got, artifact)
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

func TestSaveLoadREPLSessionRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	session := REPLSession{
		CurrentWorkspace:         "ws_1",
		FocusThread:              "th_1",
		InspectionScope:          []string{"en_1", "en_2"},
		PendingCheckpointContext: "belief=stable",
	}
	if err := store.SaveREPLSession(session); err != nil {
		t.Fatalf("SaveREPLSession() error = %v", err)
	}

	got, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if got.CurrentWorkspace != session.CurrentWorkspace || got.FocusThread != session.FocusThread {
		t.Fatalf("unexpected repl session: got %+v want %+v", got, session)
	}
	if len(got.InspectionScope) != 2 || got.PendingCheckpointContext != session.PendingCheckpointContext {
		t.Fatalf("unexpected repl session details: got %+v want %+v", got, session)
	}
}

func TestSaveLoadDerivedIndexRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, ".sandnote"))
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	index := DerivedIndex{
		GeneratedAt: time.Now().UTC().Round(time.Second),
		Threads: []DerivedThreadRecord{
			{
				ID:          "th_1",
				Question:    "How should resumability be indexed?",
				Vitality:    model.VitalityLive,
				WorkspaceID: "ws_1",
				TopicIDs:    []string{"tp_1"},
				UpdatedAt:   time.Now().UTC().Round(time.Second),
			},
		},
		Workspaces: []DerivedWorkspaceRecord{
			{
				ID:          "ws_1",
				Name:        "task/index",
				ThreadCount: 1,
				UpdatedAt:   time.Now().UTC().Round(time.Second),
			},
		},
		Topics: []DerivedTopicRecord{
			{
				ID:          "tp_1",
				Name:        "indexing",
				ThreadCount: 1,
				EntryCount:  1,
				UpdatedAt:   time.Now().UTC().Round(time.Second),
			},
		},
	}

	if err := store.SaveDerivedIndex(index); err != nil {
		t.Fatalf("SaveDerivedIndex() error = %v", err)
	}

	got, err := store.LoadDerivedIndex()
	if err != nil {
		t.Fatalf("LoadDerivedIndex() error = %v", err)
	}
	if len(got.Threads) != 1 || got.Threads[0].ID != "th_1" {
		t.Fatalf("unexpected derived index threads: %+v", got.Threads)
	}
	if len(got.Workspaces) != 1 || got.Workspaces[0].ThreadCount != 1 {
		t.Fatalf("unexpected derived index workspaces: %+v", got.Workspaces)
	}
	if len(got.Topics) != 1 || got.Topics[0].EntryCount != 1 {
		t.Fatalf("unexpected derived index topics: %+v", got.Topics)
	}
}
