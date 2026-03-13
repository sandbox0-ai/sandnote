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

func TestEntryCreateAndRevise(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "entry", "create", "--id", "en_new", "--subject", "new subject", "--meaning", "new meaning")
	output := executeCLI(t, root, "entry", "revise", "en_new", "--state", "draft", "--json")

	var got model.Entry
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.State != "draft" || got.Meaning != "new meaning" {
		t.Fatalf("unexpected entry revision: %+v", got)
	}
}

func TestWorkspaceFocusAttachesThread(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "workspace", "focus", "ws_1", "th_1", "--json")

	var got model.Workspace
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.FocusThreadID != "th_1" || len(got.ThreadIDs) != 1 {
		t.Fatalf("unexpected workspace focus state: %+v", got)
	}
}

func TestTopicOrientUpdatesOrientation(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "topic", "orient", "tp_1", "--orientation", "Start here for auth work.", "--json")

	var got model.Topic
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Orientation != "Start here for auth work." {
		t.Fatalf("unexpected topic orientation: %+v", got)
	}
}

func TestTopicPromoteFromThreadAddsAnchorByDefault(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "topic", "promote", "tp_1", "--thread", "th_1", "--json")

	var got model.Topic
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !contains(got.ThreadIDs, "th_1") {
		t.Fatalf("expected promoted thread in topic: %+v", got)
	}
	if !contains(got.EntryIDs, "en_1") {
		t.Fatalf("expected re-entry anchor in topic: %+v", got)
	}
}

func TestTopicPromoteCanIncludeSupportingEntries(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)

	store := fsstore.New(root)
	extra := model.Entry{
		ID:        "en_2",
		Subject:   "supporting",
		Meaning:   "extra supporting material",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := store.SaveEntry(extra); err != nil {
		t.Fatalf("SaveEntry() error = %v", err)
	}
	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	thread.SupportingIDs = []string{"en_1", "en_2"}
	thread.UpdatedAt = nowUTC()
	if err := store.SaveThread(thread); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	output := executeCLI(t, root, "topic", "promote", "tp_1", "--thread", "th_1", "--include-supporting", "--json")

	var got model.Topic
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !contains(got.EntryIDs, "en_2") {
		t.Fatalf("expected supporting entry in topic: %+v", got)
	}
}

func TestREPLMaintainsWorkspaceAndThreadState(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)

	in := bytes.NewBufferString(strings.Join([]string{
		"workspace use ws_1",
		"thread focus th_1",
		"resume",
		"inspect",
		"checkpoint belief=stable edge=open lean=next anchor=en_1",
		"transition dormant",
		"status",
		"exit",
	}, "\n"))
	out := &bytes.Buffer{}

	state := &replState{}
	if err := runREPL(in, out, store, state); err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"sandnote repl",
		"workspace ws_1",
		"thread th_1",
		"resume th_1",
		"supporting entries:",
		"vitality: dormant",
		"workspace: ws_1",
		"focus thread: th_1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("repl output missing %q:\n%s", want, text)
		}
	}

	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.Vitality != model.VitalityDormant || thread.CurrentBelief != "stable" || thread.ReentryAnchor != "en_1" {
		t.Fatalf("unexpected thread after repl: %+v", thread)
	}
}

func seedInteractionStore(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), ".sandnote")
	store := fsstore.New(root)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	entry := model.Entry{
		ID:        "en_1",
		Subject:   "anchor",
		Meaning:   "thread anchor",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := store.SaveEntry(entry); err != nil {
		t.Fatalf("SaveEntry() error = %v", err)
	}

	workspace := model.Workspace{
		ID:        "ws_1",
		Name:      "task/auth",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := store.SaveWorkspace(workspace); err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}

	topic := model.Topic{
		ID:        "tp_1",
		Name:      "auth",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := store.SaveTopic(topic); err != nil {
		t.Fatalf("SaveTopic() error = %v", err)
	}

	thread := model.Thread{
		ID:            "th_1",
		Question:      "How should auth threads resume?",
		CurrentBelief: "resume is the default continuation surface",
		OpenEdge:      "need a better checkpoint path",
		NextLean:      "inspect supporting entries",
		ReentryAnchor: "en_1",
		Vitality:      model.VitalityLive,
		SupportingIDs: []string{"en_1"},
		CreatedAt:     nowUTC(),
		UpdatedAt:     nowUTC(),
	}
	if err := store.SaveThread(thread); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	return root
}
