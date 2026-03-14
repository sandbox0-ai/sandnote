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

func TestCanonicalWorkflowPreservesTopicReentryAfterThreadSettles(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), ".sandnote")
	executeCLI(t, root, "init")
	executeCLI(t, root, "workspace", "create", "--id", "ws_auth", "--name", "task/auth")
	executeCLI(t, root, "topic", "create", "--id", "tp_auth", "--name", "auth", "--orientation", "Start with the auth thread.")
	executeCLI(t, root, "entry", "create", "--id", "en_auth", "--subject", "auth anchor", "--meaning", "resume auth work here")
	executeCLI(t, root, "thread", "create", "--id", "th_auth", "--question", "How should auth work continue?", "--workspace", "ws_auth")
	executeCLI(t, root, "entry", "attach", "en_auth", "--thread", "th_auth", "--topic", "tp_auth")
	executeCLI(t, root, "workspace", "focus", "ws_auth", "th_auth")
	executeCLI(t, root, "workspace", "use", "ws_auth")

	resumeOutput := executeCLI(t, root, "resume", "--json")
	var resumeGot resumeView
	if err := json.Unmarshal(resumeOutput.Bytes(), &resumeGot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if resumeGot.Thread.ID != "th_auth" || resumeGot.WorkspaceID != "ws_auth" {
		t.Fatalf("unexpected resumed workflow state: %+v", resumeGot)
	}

	executeCLI(
		t,
		root,
		"thread", "checkpoint", "th_auth",
		"--belief", "auth-flow-is-working",
		"--open-edge", "promote-the-durable-auth-understanding",
		"--next-lean", "promote-auth-topic",
		"--reentry-anchor", "en_auth",
	)
	executeCLI(t, root, "topic", "promote", "tp_auth", "--thread", "th_auth", "--include-supporting")

	topicEntriesOutput := executeCLI(t, root, "topic", "entries", "tp_auth", "--json")
	var topicEntries []model.Entry
	if err := json.Unmarshal(topicEntriesOutput.Bytes(), &topicEntries); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(topicEntries) != 1 || topicEntries[0].ID != "en_auth" {
		t.Fatalf("unexpected promoted topic entries: %+v", topicEntries)
	}

	topicThreadsOutput := executeCLI(t, root, "topic", "threads", "tp_auth", "--json")
	var topicThreads []threadListItem
	if err := json.Unmarshal(topicThreadsOutput.Bytes(), &topicThreads); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(topicThreads) != 1 || topicThreads[0].ID != "th_auth" {
		t.Fatalf("unexpected promoted topic threads: %+v", topicThreads)
	}

	executeCLI(t, root, "thread", "transition", "th_auth", "--to", "settled")

	store := fsstore.New(root)
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.CurrentWorkspace != "ws_auth" || session.FocusThread != "" {
		t.Fatalf("unexpected session after settling thread: %+v", session)
	}

	_, err = executeCLIAllowError(root, "resume")
	if err == nil || !strings.Contains(err.Error(), "no live threads") {
		t.Fatalf("expected resume to fail once all threads settled, got %v", err)
	}

	topic, err := store.LoadTopic("tp_auth")
	if err != nil {
		t.Fatalf("LoadTopic() error = %v", err)
	}
	if !contains(topic.EntryIDs, "en_auth") || !contains(topic.ThreadIDs, "th_auth") {
		t.Fatalf("expected topic re-entry surface preserved after settling thread: %+v", topic)
	}
}

func TestWorkflowRestoresREPLStateAcrossRestart(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), ".sandnote")
	executeCLI(t, root, "init")
	executeCLI(t, root, "workspace", "create", "--id", "ws_auth", "--name", "task/auth")
	executeCLI(t, root, "entry", "create", "--id", "en_auth", "--subject", "auth anchor", "--meaning", "resume auth work here")
	executeCLI(t, root, "thread", "create", "--id", "th_auth", "--question", "How should auth work continue?", "--workspace", "ws_auth")
	executeCLI(t, root, "entry", "attach", "en_auth", "--thread", "th_auth")
	executeCLI(t, root, "workspace", "focus", "ws_auth", "th_auth")
	executeCLI(t, root, "workspace", "use", "ws_auth")

	store := fsstore.New(root)
	firstIn := bytes.NewBufferString(strings.Join([]string{
		"resume",
		"inspect",
		"checkpoint belief=resume-confirmed edge=needs-promotion lean=promote-topic anchor=en_auth",
		"exit",
	}, "\n"))
	firstOut := &bytes.Buffer{}
	firstState, err := loadREPLState(store)
	if err != nil {
		t.Fatalf("loadREPLState() error = %v", err)
	}
	if err := runREPL(firstIn, firstOut, store, firstState); err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	thread, err := store.LoadThread("th_auth")
	if err != nil {
		t.Fatalf("LoadThread() error = %v", err)
	}
	if thread.CurrentBelief != "resume-confirmed" || thread.OpenEdge != "needs-promotion" || thread.ReentryAnchor != "en_auth" {
		t.Fatalf("unexpected thread after repl checkpoint: %+v", thread)
	}

	secondIn := bytes.NewBufferString("status\nresume\nexit\n")
	secondOut := &bytes.Buffer{}
	secondState, err := loadREPLState(store)
	if err != nil {
		t.Fatalf("loadREPLState() error = %v", err)
	}
	if err := runREPL(secondIn, secondOut, store, secondState); err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	text := secondOut.String()
	for _, want := range []string{
		"workspace: ws_auth",
		"focus thread: th_auth",
		"inspection scope: en_auth",
		"resume th_auth",
		"belief: resume-confirmed",
		"open edge: needs-promotion",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected restarted repl output to contain %q:\n%s", want, text)
		}
	}
}

func TestWorkflowRetargetsToReplacementThreadAcrossSessions(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), ".sandnote")
	executeCLI(t, root, "init")
	executeCLI(t, root, "workspace", "create", "--id", "ws_auth", "--name", "task/auth")
	executeCLI(t, root, "entry", "create", "--id", "en_primary", "--subject", "primary anchor", "--meaning", "primary auth thread")
	executeCLI(t, root, "entry", "create", "--id", "en_secondary", "--subject", "secondary anchor", "--meaning", "secondary auth thread")
	executeCLI(t, root, "thread", "create", "--id", "th_primary", "--question", "How should primary auth work continue?", "--workspace", "ws_auth")
	executeCLI(t, root, "thread", "create", "--id", "th_secondary", "--question", "How should fallback auth work continue?", "--workspace", "ws_auth")
	executeCLI(t, root, "entry", "attach", "en_primary", "--thread", "th_primary")
	executeCLI(t, root, "entry", "attach", "en_secondary", "--thread", "th_secondary")
	executeCLI(
		t,
		root,
		"thread", "checkpoint", "th_primary",
		"--belief", "primary-thread-is-active",
		"--open-edge", "primary-needs-a-transition",
		"--next-lean", "hand-off-to-secondary",
		"--reentry-anchor", "en_primary",
	)
	executeCLI(
		t,
		root,
		"thread", "checkpoint", "th_secondary",
		"--belief", "secondary-thread-is-ready",
		"--open-edge", "continue-through-secondary",
		"--next-lean", "resume-secondary",
		"--reentry-anchor", "en_secondary",
	)
	executeCLI(t, root, "workspace", "focus", "ws_auth", "th_primary")
	executeCLI(t, root, "workspace", "use", "ws_auth")
	executeCLI(t, root, "thread", "transition", "th_primary", "--to", "dormant")

	store := fsstore.New(root)
	session, err := store.LoadREPLSession()
	if err != nil {
		t.Fatalf("LoadREPLSession() error = %v", err)
	}
	if session.FocusThread != "th_secondary" || session.CurrentWorkspace != "ws_auth" {
		t.Fatalf("unexpected session after replacement retarget: %+v", session)
	}

	output := executeCLI(t, root, "resume", "--json")
	var got resumeView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Thread.ID != "th_secondary" {
		t.Fatalf("expected replacement thread to become resumable default: %+v", got)
	}

	in := bytes.NewBufferString("status\nresume\nexit\n")
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
		"focus thread: th_secondary",
		"resume th_secondary",
		"belief: secondary-thread-is-ready",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected restarted repl to stay on replacement thread %q:\n%s", want, text)
		}
	}
}

func executeCLIAllowError(root string, args ...string) (*bytes.Buffer, error) {
	cmd := NewRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs(append([]string{"--root", root}, args...))
	err := cmd.Execute()
	if err != nil {
		return output, err
	}
	return output, nil
}
