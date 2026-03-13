package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/sandbox0-ai/sandnote/internal/model"
	"github.com/sandbox0-ai/sandnote/internal/store/fsstore"
	"github.com/spf13/cobra"
)

type replState struct {
	currentWorkspace         string
	focusThread              string
	inspectionScope          []string
	pendingCheckpointContext string
}

func newREPLCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Start the stateful working console",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := requireStore(opts.storeRoot)
			if err != nil {
				return err
			}
			state := &replState{}
			return runREPL(cmd.InOrStdin(), cmd.OutOrStdout(), store, state)
		},
	}
}

func runREPL(in io.Reader, out io.Writer, store *fsstore.Store, state *replState) error {
	scanner := bufio.NewScanner(in)
	fmt.Fprintln(out, "sandnote repl")
	fmt.Fprintln(out, "type 'help' for commands, 'exit' to quit")

	for {
		fmt.Fprint(out, "sandnote> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		args := strings.Fields(line)
		switch args[0] {
		case "exit", "quit":
			return nil
		case "help":
			fmt.Fprintln(out, replHelp())
		case "status":
			fmt.Fprint(out, formatREPLStatus(*state))
		case "workspace":
			if err := replWorkspace(out, store, state, args[1:]); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		case "thread":
			if err := replThread(out, store, state, args[1:]); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		case "inspect":
			if err := replInspect(out, store, state); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		case "resume":
			if err := replResume(out, store, state); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		case "checkpoint":
			if err := replCheckpoint(out, store, state, strings.TrimSpace(strings.TrimPrefix(line, "checkpoint"))); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		case "transition":
			if len(args) < 2 {
				fmt.Fprintln(out, "error: transition requires a vitality state")
				continue
			}
			if err := replTransition(out, store, state, args[1]); err != nil {
				fmt.Fprintf(out, "error: %v\n", err)
			}
		default:
			fmt.Fprintf(out, "error: unknown command %q\n", args[0])
		}
	}
}

func replHelp() string {
	return joinLines(
		"commands:",
		"  status",
		"  workspace use <id>",
		"  workspace show",
		"  thread focus <id>",
		"  thread show",
		"  resume",
		"  inspect",
		"  checkpoint belief=<text> edge=<text> lean=<text> anchor=<id>",
		"  transition <live|dormant|settled>",
		"  exit",
	)
}

func formatREPLStatus(state replState) string {
	return joinLines(
		optionalLabel("workspace", state.currentWorkspace),
		optionalLabel("focus thread", state.focusThread),
		optionalLabel("inspection scope", strings.Join(state.inspectionScope, ", ")),
		optionalLabel("pending checkpoint context", state.pendingCheckpointContext),
	)
}

func replWorkspace(out io.Writer, store *fsstore.Store, state *replState, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("workspace subcommand required")
	}
	switch args[0] {
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: workspace use <id>")
		}
		workspace, err := store.LoadWorkspace(args[1])
		if err != nil {
			return err
		}
		state.currentWorkspace = workspace.ID
		state.focusThread = workspace.FocusThreadID
		fmt.Fprint(out, formatWorkspace(workspace))
		return nil
	case "show":
		if state.currentWorkspace == "" {
			return fmt.Errorf("no workspace selected")
		}
		workspace, err := store.LoadWorkspace(state.currentWorkspace)
		if err != nil {
			return err
		}
		fmt.Fprint(out, formatWorkspace(workspace))
		return nil
	default:
		return fmt.Errorf("unknown workspace subcommand %q", args[0])
	}
}

func replThread(out io.Writer, store *fsstore.Store, state *replState, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("thread subcommand required")
	}
	switch args[0] {
	case "focus":
		if len(args) != 2 {
			return fmt.Errorf("usage: thread focus <id>")
		}
		thread, err := store.LoadThread(args[1])
		if err != nil {
			return err
		}
		state.focusThread = thread.ID
		state.pendingCheckpointContext = ""
		fmt.Fprint(out, formatThreadShow(thread))
		return nil
	case "show":
		if state.focusThread == "" {
			return fmt.Errorf("no focus thread selected")
		}
		thread, err := store.LoadThread(state.focusThread)
		if err != nil {
			return err
		}
		fmt.Fprint(out, formatThreadShow(thread))
		return nil
	default:
		return fmt.Errorf("unknown thread subcommand %q", args[0])
	}
}

func replResume(out io.Writer, store *fsstore.Store, state *replState) error {
	thread, err := focusedThread(store, state)
	if err != nil {
		return err
	}
	state.inspectionScope = []string{thread.ReentryAnchor}
	fmt.Fprint(out, formatThreadResume(thread))
	return nil
}

func replInspect(out io.Writer, store *fsstore.Store, state *replState) error {
	thread, err := focusedThread(store, state)
	if err != nil {
		return err
	}
	entries, err := store.LoadEntries(thread.SupportingIDs)
	if err != nil {
		return err
	}
	state.inspectionScope = thread.SupportingIDs
	fmt.Fprint(out, formatThreadInspect(thread, entries))
	return nil
}

func replCheckpoint(out io.Writer, store *fsstore.Store, state *replState, payload string) error {
	thread, err := focusedThread(store, state)
	if err != nil {
		return err
	}
	updates := parseCheckpointPayload(payload)
	if len(updates) == 0 {
		return fmt.Errorf("checkpoint requires belief=, edge=, lean=, or anchor=")
	}
	if value, ok := updates["belief"]; ok {
		thread.CurrentBelief = value
	}
	if value, ok := updates["edge"]; ok {
		thread.OpenEdge = value
	}
	if value, ok := updates["lean"]; ok {
		thread.NextLean = value
	}
	if value, ok := updates["anchor"]; ok {
		thread.ReentryAnchor = value
	}
	thread.UpdatedAt = nowUTC()
	if err := store.SaveThread(thread); err != nil {
		return err
	}
	state.pendingCheckpointContext = ""
	fmt.Fprint(out, formatThreadResume(thread))
	return nil
}

func replTransition(out io.Writer, store *fsstore.Store, state *replState, vitality string) error {
	thread, err := focusedThread(store, state)
	if err != nil {
		return err
	}
	next := model.VitalityState(vitality)
	if err := next.Validate(); err != nil {
		return err
	}
	thread.Vitality = next
	thread.UpdatedAt = nowUTC()
	if err := store.SaveThread(thread); err != nil {
		return err
	}
	fmt.Fprint(out, formatThreadShow(thread))
	return nil
}

func focusedThread(store *fsstore.Store, state *replState) (model.Thread, error) {
	if state.focusThread == "" {
		return model.Thread{}, fmt.Errorf("no focus thread selected")
	}
	return store.LoadThread(state.focusThread)
}

func parseCheckpointPayload(payload string) map[string]string {
	fields := strings.Fields(strings.TrimSpace(payload))
	out := make(map[string]string, len(fields))
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	return out
}
