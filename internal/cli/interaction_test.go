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

func TestEntryLinkAddsRelatedContextWithoutDuplicates(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "entry", "link", "en_1", "th_1", "tp_1", "th_1", "--json")

	var got model.Entry
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.RelatedContext) != 2 || !contains(got.RelatedContext, "th_1") || !contains(got.RelatedContext, "tp_1") {
		t.Fatalf("unexpected related context after link: %+v", got)
	}

	store := fsstore.New(root)
	entry, err := store.LoadEntry("en_1")
	if err != nil {
		t.Fatalf("LoadEntry() error = %v", err)
	}
	if len(entry.RelatedContext) != 2 {
		t.Fatalf("expected related context persisted without duplicates: %+v", entry)
	}
}

func TestEntryAttachAddsThreadAndTopicRelations(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "entry", "attach", "en_1", "--thread", "th_1", "--topic", "tp_1", "--json")

	var got model.Entry
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !contains(got.RelatedContext, "th_1") || !contains(got.RelatedContext, "tp_1") {
		t.Fatalf("unexpected related context after attach: %+v", got)
	}

	store := fsstore.New(root)
	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if len(thread.SupportingIDs) != 1 || thread.SupportingIDs[0] != "en_1" {
		t.Fatalf("expected entry attached to thread support context: %+v", thread)
	}

	topic, err := store.LoadTopic("tp_1")
	if err != nil {
		t.Fatalf("LoadTopic() error = %v", err)
	}
	if len(topic.EntryIDs) != 1 || topic.EntryIDs[0] != "en_1" {
		t.Fatalf("expected entry attached to topic: %+v", topic)
	}
}

func TestEntryAttachAvoidsDuplicateTargets(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "entry", "attach", "en_1", "--thread", "th_1", "--topic", "tp_1")
	executeCLI(t, root, "entry", "attach", "en_1", "--thread", "th_1", "--topic", "tp_1")

	store := fsstore.New(root)
	entry, err := store.LoadEntry("en_1")
	if err != nil {
		t.Fatalf("LoadEntry() error = %v", err)
	}
	if len(entry.RelatedContext) != 2 {
		t.Fatalf("expected deduped related context after repeated attach: %+v", entry)
	}

	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if len(thread.SupportingIDs) != 1 {
		t.Fatalf("expected deduped supporting ids after repeated attach: %+v", thread)
	}

	topic, err := store.LoadTopic("tp_1")
	if err != nil {
		t.Fatalf("LoadTopic() error = %v", err)
	}
	if len(topic.EntryIDs) != 1 {
		t.Fatalf("expected deduped topic entry ids after repeated attach: %+v", topic)
	}
}

func TestEntryArchiveMarksEntryArchivedWithoutLosingContent(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "entry", "link", "en_1", "th_1", "tp_1")
	output := executeCLI(t, root, "entry", "archive", "en_1", "--json")

	var got model.Entry
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.State != "archived" || got.Subject != "anchor" || got.Meaning != "thread anchor" {
		t.Fatalf("unexpected archived entry: %+v", got)
	}
	if len(got.RelatedContext) != 2 {
		t.Fatalf("expected related context preserved on archive: %+v", got)
	}
}

func TestEntryArchiveRemainsVisibleInShowOutput(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "entry", "archive", "en_1")
	output := executeCLI(t, root, "entry", "show", "en_1")
	text := output.String()

	for _, want := range []string{
		"entry en_1",
		"state: archived",
		"meaning: thread anchor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected archived entry to remain visible, missing %q:\n%s", want, text)
		}
	}
}

func TestEntryShowDisplaysRelatedContext(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "entry", "link", "en_1", "th_1", "tp_1")
	output := executeCLI(t, root, "entry", "show", "en_1")
	text := output.String()

	if !strings.Contains(text, "related: th_1, tp_1") {
		t.Fatalf("expected related context in entry show output:\n%s", text)
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

func TestThreadCreateWithWorkspaceKeepsWorkspaceShowAndThreadsConsistent(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "workspace", "create", "--id", "ws_auth", "--name", "task/auth")
	executeCLI(t, root, "thread", "create", "--id", "th_auth", "--question", "How should auth work continue?", "--workspace", "ws_auth")

	showOutput := executeCLI(t, root, "workspace", "show", "ws_auth", "--json")
	var workspace model.Workspace
	if err := json.Unmarshal(showOutput.Bytes(), &workspace); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !contains(workspace.ThreadIDs, "th_auth") {
		t.Fatalf("expected workspace show to include derived thread membership: %+v", workspace)
	}

	threadsOutput := executeCLI(t, root, "workspace", "threads", "ws_auth", "--json")
	var threads []threadListItem
	if err := json.Unmarshal(threadsOutput.Bytes(), &threads); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(threads) != 1 || threads[0].ID != "th_auth" {
		t.Fatalf("expected workspace threads to match workspace show: %+v", threads)
	}

	store := fsstore.New(root)
	persisted, err := store.LoadWorkspace("ws_auth")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	if !contains(persisted.ThreadIDs, "th_auth") {
		t.Fatalf("expected persisted workspace membership to stay aligned: %+v", persisted)
	}
}

func TestWorkspaceAttachAssignsMembershipWithoutChangingFocus(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)
	now := nowUTC()
	free := model.Thread{
		ID:        "th_free",
		Question:  "What should get attached next?",
		Vitality:  model.VitalityLive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.SaveThread(free); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	output := executeCLI(t, root, "workspace", "attach", "ws_1", "th_free", "--json")

	var got model.Workspace
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.FocusThreadID != "" || !contains(got.ThreadIDs, "th_free") {
		t.Fatalf("unexpected workspace after attach: %+v", got)
	}

	thread, err := store.LoadThread("th_free")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.WorkspaceID != "ws_1" {
		t.Fatalf("expected thread attached to workspace: %+v", thread)
	}
}

func TestWorkspaceDetachClearsMembershipAndFocusedSessionWithoutReplacement(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "workspace", "focus", "ws_1", "th_1")
	executeCLI(t, root, "workspace", "use", "ws_1")
	executeCLI(t, root, "workspace", "detach", "ws_1", "th_1")

	store := fsstore.New(root)
	workspace, err := store.LoadWorkspace("ws_1")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	if workspace.FocusThreadID != "" || contains(workspace.ThreadIDs, "th_1") {
		t.Fatalf("expected workspace membership cleared after detach: %+v", workspace)
	}

	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.WorkspaceID != "" {
		t.Fatalf("expected thread workspace cleared after detach: %+v", thread)
	}

	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_1" || session.FocusThread != "" {
		t.Fatalf("unexpected session after detach: %+v", session)
	}
}

func TestWorkspaceDetachRetargetsFocusAndSessionToReplacement(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)
	now := nowUTC()
	other := model.Thread{
		ID:            "th_2",
		Question:      "What should replace the detached focus?",
		CurrentBelief: "keep a second live thread available",
		OpenEdge:      "pick the next active thread",
		ReentryAnchor: "en_1",
		Vitality:      model.VitalityLive,
		WorkspaceID:   "ws_1",
		SupportingIDs: []string{"en_1"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.SaveThread(other); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}
	workspace, err := store.LoadWorkspace("ws_1")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	workspace.ThreadIDs = []string{"th_1", "th_2"}
	workspace.UpdatedAt = nowUTC()
	if err := store.SaveWorkspace(workspace); err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}

	executeCLI(t, root, "workspace", "focus", "ws_1", "th_1")
	executeCLI(t, root, "workspace", "use", "ws_1")
	executeCLI(t, root, "workspace", "detach", "ws_1", "th_1")

	workspace, err = store.LoadWorkspace("ws_1")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	if workspace.FocusThreadID != "th_2" || contains(workspace.ThreadIDs, "th_1") {
		t.Fatalf("expected workspace focus retargeted after detach: %+v", workspace)
	}

	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_1" || session.FocusThread != "th_2" {
		t.Fatalf("unexpected session after detach retarget: %+v", session)
	}
}

func TestThreadTransitionClearsFocusWhenNoLiveReplacementExists(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "workspace", "focus", "ws_1", "th_1")
	executeCLI(t, root, "workspace", "use", "ws_1")
	executeCLI(t, root, "thread", "transition", "th_1", "--to", "dormant")

	store := fsstore.New(root)
	workspace, err := store.LoadWorkspace("ws_1")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	if workspace.FocusThreadID != "" {
		t.Fatalf("expected workspace focus cleared: %+v", workspace)
	}
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.FocusThread != "" || session.CurrentWorkspace != "ws_1" {
		t.Fatalf("unexpected session after clearing focus: %+v", session)
	}
}

func TestThreadTransitionRetargetsFocusToAnotherLiveThread(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)
	now := nowUTC()
	other := model.Thread{
		ID:            "th_2",
		Question:      "How should auth work continue next?",
		CurrentBelief: "keep a second live thread available",
		OpenEdge:      "promote a replacement focus",
		ReentryAnchor: "en_1",
		Vitality:      model.VitalityLive,
		WorkspaceID:   "ws_1",
		SupportingIDs: []string{"en_1"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.SaveThread(other); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	executeCLI(t, root, "workspace", "focus", "ws_1", "th_1")
	executeCLI(t, root, "workspace", "use", "ws_1")
	executeCLI(t, root, "thread", "transition", "th_1", "--to", "dormant")

	workspace, err := store.LoadWorkspace("ws_1")
	if err != nil {
		t.Fatalf("LoadWorkspace() error = %v", err)
	}
	if workspace.FocusThreadID != "th_2" {
		t.Fatalf("expected workspace focus retargeted to th_2: %+v", workspace)
	}
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.FocusThread != "th_2" || session.CurrentWorkspace != "ws_1" {
		t.Fatalf("unexpected session after retargeting focus: %+v", session)
	}
}

func TestWorkspaceUsePersistsActiveSelection(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "workspace", "focus", "ws_1", "th_1")
	executeCLI(t, root, "workspace", "use", "ws_1")

	store := fsstore.New(root)
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_1" || session.FocusThread != "th_1" {
		t.Fatalf("unexpected session after workspace use: %+v", session)
	}
}

func TestThreadFocusPersistsActiveSelection(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "thread", "focus", "th_1")

	store := fsstore.New(root)
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_1" || session.FocusThread != "th_1" {
		t.Fatalf("unexpected session after thread focus: %+v", session)
	}
}

func TestThreadFocusClearsStaleWorkspaceWhenThreadIsUnscoped(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)
	now := nowUTC()
	free := model.Thread{
		ID:        "th_free",
		Question:  "What should continue without workspace context?",
		Vitality:  model.VitalityLive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.SaveThread(free); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	executeCLI(t, root, "workspace", "use", "ws_1")
	executeCLI(t, root, "thread", "focus", "th_free")

	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "" || session.FocusThread != "th_free" {
		t.Fatalf("expected unscoped thread focus to clear stale workspace: %+v", session)
	}
}

func TestThreadAttachAndDetachKeepsEntryRelatedContextAligned(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)

	executeCLI(t, root, "thread", "attach", "th_1", "en_1")

	store := fsstore.New(root)
	entry, err := store.LoadEntry("en_1")
	if err != nil {
		t.Fatalf("LoadEntry() error = %v", err)
	}
	if !contains(entry.RelatedContext, "th_1") {
		t.Fatalf("expected thread attach to update entry related context: %+v", entry)
	}

	executeCLI(t, root, "thread", "detach", "th_1", "en_1")

	entry, err = store.LoadEntry("en_1")
	if err != nil {
		t.Fatalf("LoadEntry() error = %v", err)
	}
	if contains(entry.RelatedContext, "th_1") {
		t.Fatalf("expected thread detach to remove entry related context: %+v", entry)
	}
}

func TestTopLevelResumePersistsSelectedThread(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "resume")

	store := fsstore.New(root)
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_1" || session.FocusThread != "th_1" {
		t.Fatalf("unexpected session after resume: %+v", session)
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

func TestTopicEntriesListsAttachedEntries(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "topic", "entries", "tp_1", "--json")

	var got []model.Entry
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "en_1" {
		t.Fatalf("unexpected topic entries: %+v", got)
	}
}

func TestTopicThreadsListsAttachedThreads(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	output := executeCLI(t, root, "topic", "threads", "tp_1", "--json")

	var got []threadListItem
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "th_1" || got[0].WorkspaceID != "ws_1" {
		t.Fatalf("unexpected topic threads: %+v", got)
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
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("repl output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "focus thread: th_1") {
		t.Fatalf("repl status should not keep a dormant thread focused:\n%s", text)
	}

	thread, err := store.LoadThread("th_1")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.Vitality != model.VitalityDormant || thread.CurrentBelief != "stable" || thread.ReentryAnchor != "en_1" {
		t.Fatalf("unexpected thread after repl: %+v", thread)
	}
}

func TestREPLRestoresPersistedSessionState(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)
	if err := store.SaveREPLSession(fsstore.REPLSession{
		CurrentWorkspace:         "ws_1",
		FocusThread:              "th_1",
		InspectionScope:          []string{"en_1"},
		PendingCheckpointContext: "belief=carry",
	}); err != nil {
		t.Fatalf("SaveREPLSession() error = %v", err)
	}

	in := bytes.NewBufferString("status\nexit\n")
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
		"workspace: ws_1",
		"focus thread: th_1",
		"inspection scope: en_1",
		"pending checkpoint context: belief=carry",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("restored status missing %q:\n%s", want, text)
		}
	}
}

func TestREPLCheckpointRejectsLowQualityLiveCheckpoint(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	store := fsstore.New(root)

	bare := model.Thread{
		ID:          "th_bare",
		Question:    "What if the repl tries a weak checkpoint?",
		Vitality:    model.VitalityLive,
		WorkspaceID: "ws_1",
		CreatedAt:   nowUTC(),
		UpdatedAt:   nowUTC(),
	}
	if err := store.SaveThread(bare); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}
	if err := store.SaveREPLSession(fsstore.REPLSession{
		CurrentWorkspace: "ws_1",
		FocusThread:      "th_bare",
	}); err != nil {
		t.Fatalf("SaveREPLSession() error = %v", err)
	}

	in := bytes.NewBufferString("checkpoint belief=still-thinking\nexit\n")
	out := &bytes.Buffer{}

	state, err := loadREPLState(store)
	if err != nil {
		t.Fatalf("loadREPLState() error = %v", err)
	}
	if err := runREPL(in, out, store, state); err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	if !strings.Contains(out.String(), "live thread checkpoints must leave a clear") {
		t.Fatalf("expected repl checkpoint quality error:\n%s", out.String())
	}
}

func TestIndexRebuildSupportsThreadWorkspaceAndTopicQueries(t *testing.T) {
	t.Parallel()

	root := seedInteractionStore(t)
	executeCLI(t, root, "index", "rebuild")

	threadOutput := executeCLI(t, root, "thread", "list", "--workspace", "ws_1", "--topic", "tp_1", "--query", "checkpoint", "--json")
	var threads []threadListItem
	if err := json.Unmarshal(threadOutput.Bytes(), &threads); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(threads) != 1 || threads[0].ID != "th_1" {
		t.Fatalf("unexpected thread query results: %+v", threads)
	}

	workspaceOutput := executeCLI(t, root, "workspace", "list", "--query", "task/auth", "--json")
	var workspaces []workspaceListItem
	if err := json.Unmarshal(workspaceOutput.Bytes(), &workspaces); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(workspaces) != 1 || workspaces[0].ThreadCount != 1 {
		t.Fatalf("unexpected workspace query results: %+v", workspaces)
	}

	topicOutput := executeCLI(t, root, "topic", "list", "--query", "auth", "--json")
	var topics []topicListItem
	if err := json.Unmarshal(topicOutput.Bytes(), &topics); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(topics) != 1 || topics[0].ThreadCount != 1 || topics[0].EntryCount != 1 {
		t.Fatalf("unexpected topic query results: %+v", topics)
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
		EntryIDs:  []string{"en_1"},
		ThreadIDs: []string{"th_1"},
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
		WorkspaceID:   "ws_1",
		SupportingIDs: []string{"en_1"},
		CreatedAt:     nowUTC(),
		UpdatedAt:     nowUTC(),
	}
	if err := store.SaveThread(thread); err != nil {
		t.Fatalf("SaveThread() error = %v", err)
	}

	return root
}
