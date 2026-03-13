package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
)

func TestTopLevelResumeUsesFocusedThread(t *testing.T) {
	t.Parallel()

	root := seedFrontierStore(t)
	output := executeCLI(t, root, "resume", "--json")

	var got resumeView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Thread.ID != "th_focus" {
		t.Fatalf("unexpected resumed thread: %+v", got)
	}
	if got.ContinuationPressure < 100 {
		t.Fatalf("expected focused thread to get high continuation pressure: %+v", got)
	}
	if !contains(got.Reasons, "focused thread") {
		t.Fatalf("expected focused-thread reason: %+v", got)
	}
}

func TestThreadFrontierPrefersWorkspaceAndReturnsRankedLiveThreads(t *testing.T) {
	t.Parallel()

	root := seedFrontierStore(t)
	output := executeCLI(t, root, "thread", "frontier", "--workspace", "ws_auth", "--json")

	var got []frontierItem
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected frontier count: %+v", got)
	}
	if got[0].ID != "th_focus" {
		t.Fatalf("expected focused auth thread first: %+v", got)
	}
	for _, item := range got {
		if item.WorkspaceID != "ws_auth" {
			t.Fatalf("unexpected workspace in frontier item: %+v", item)
		}
	}
}

func TestREPLStartupAndResumeUseLiveFrontier(t *testing.T) {
	t.Parallel()

	root := seedFrontierStoreWithoutFocus(t)
	store := fsstore.New(root)

	in := bytes.NewBufferString("resume\nstatus\nexit\n")
	out := &bytes.Buffer{}

	state, err := loadREPLState(store)
	if err != nil {
		t.Fatalf("loadREPLState() error = %v", err)
	}
	if err := runREPL(in, out, store, state); err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"frontier",
		"th_focus",
		"resume th_focus",
		"focus thread: th_focus",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("repl output missing %q:\n%s", want, text)
		}
	}
}

func seedFrontierStore(t *testing.T) string {
	t.Helper()

	root := seedFrontierStoreWithoutFocus(t)
	store := fsstore.New(root)
	if err := store.SaveREPLSession(fsstore.REPLSession{
		CurrentWorkspace: "ws_auth",
		FocusThread:      "th_focus",
	}); err != nil {
		t.Fatalf("SaveREPLSession() error = %v", err)
	}
	return root
}

func seedFrontierStoreWithoutFocus(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), ".sandnote")
	store := fsstore.New(root)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := nowUTC()
	for _, entry := range []model.Entry{
		{ID: "en_auth", Subject: "auth anchor", Meaning: "resume auth work here", CreatedAt: now, UpdatedAt: now},
		{ID: "en_auth_2", Subject: "auth edge", Meaning: "supporting auth context", CreatedAt: now, UpdatedAt: now},
		{ID: "en_bill", Subject: "billing anchor", Meaning: "resume billing work here", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.SaveEntry(entry); err != nil {
			t.Fatalf("SaveEntry() error = %v", err)
		}
	}

	for _, workspace := range []model.Workspace{
		{ID: "ws_auth", Name: "task/auth", FocusThreadID: "th_focus", ThreadIDs: []string{"th_focus", "th_auth_2"}, CreatedAt: now, UpdatedAt: now},
		{ID: "ws_bill", Name: "task/billing", FocusThreadID: "th_bill", ThreadIDs: []string{"th_bill"}, CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.SaveWorkspace(workspace); err != nil {
			t.Fatalf("SaveWorkspace() error = %v", err)
		}
	}

	for _, thread := range []model.Thread{
		{
			ID:            "th_focus",
			Question:      "How should auth work resume?",
			CurrentBelief: "auth resume should stay thread-first",
			OpenEdge:      "need a stronger frontier ranking",
			NextLean:      "inspect auth supporting entries",
			ReentryAnchor: "en_auth",
			Vitality:      model.VitalityLive,
			WorkspaceID:   "ws_auth",
			SupportingIDs: []string{"en_auth", "en_auth_2"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "th_auth_2",
			Question:      "Should auth topic promotion be stricter?",
			CurrentBelief: "promotion should stay selective",
			OpenEdge:      "compare more topic entry shapes",
			ReentryAnchor: "en_auth_2",
			Vitality:      model.VitalityLive,
			WorkspaceID:   "ws_auth",
			SupportingIDs: []string{"en_auth_2"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "th_bill",
			Question:      "How should billing work resume?",
			CurrentBelief: "billing can use the same frontier model",
			OpenEdge:      "port auth heuristics",
			NextLean:      "compare workspace weighting",
			ReentryAnchor: "en_bill",
			Vitality:      model.VitalityLive,
			WorkspaceID:   "ws_bill",
			SupportingIDs: []string{"en_bill"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:        "th_old",
			Question:  "Dormant old thread",
			Vitality:  model.VitalityDormant,
			CreatedAt: now,
			UpdatedAt: now,
		},
	} {
		if err := store.SaveThread(thread); err != nil {
			t.Fatalf("SaveThread() error = %v", err)
		}
	}

	if err := store.SaveREPLSession(fsstore.REPLSession{
		CurrentWorkspace: "ws_auth",
	}); err != nil {
		t.Fatalf("SaveREPLSession() error = %v", err)
	}

	return root
}
