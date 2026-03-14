# sandnote

**A CLI-first notebook engine for agents, backed by the filesystem.**

Sandnote is a resumability-first notebook for agents.

It is not agent memory. It is the agent's notebook.

Its job is to preserve the continuity of thought so interrupted work can resume without starting over.

## Product Promise

**Come back to unfinished thought without starting over.**

For v0, the more operational version is:

**Stop without losing your place.**

## Core Model

- `entry`: a lightweight thinking unit
- `thread`: a continuable line of thought
- `workspace`: the current thinking context
- `topic`: a durable re-entry surface

These layers stay distinct:

- `entry` captures local thought
- `thread` is the main working unit
- `workspace` explains current relevance
- `topic` preserves understanding worth re-entering later

## Why Filesystem-Backed

Sandnote uses the filesystem as the source of truth:

- state is stored as plain object files
- derived indexes are rebuildable
- snapshots and restores can rely on the underlying volume
- raw files remain inspectable without introducing a database as authority

This keeps the notebook durable and operationally simple while still allowing higher-level notebook semantics.

## Current V0 Surface

Canonical CLI:

```text
sandnote entry ...
sandnote thread ...
sandnote workspace ...
sandnote topic ...
sandnote repl
```

Current core flows:

- thread-first resume and frontier selection
- checkpoint and vitality transitions
- workspace focus and active selection persistence
- topic promotion and topic re-entry reads
- stateful REPL over persisted notebook state

## Install

Build a local binary:

```bash
go build ./cmd/sandnote
```

Or install it into your Go bin directory:

```bash
go install ./cmd/sandnote
```

For a preview build with explicit metadata:

```bash
go build -ldflags "-X github.com/sandbox0-ai/sandnote/internal/cli.Version=v0.1.0-preview -X github.com/sandbox0-ai/sandnote/internal/cli.GitCommit=$(git rev-parse --short HEAD) -X github.com/sandbox0-ai/sandnote/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/sandnote
```

Inspect the current build:

```bash
sandnote version
```

## Quickstart

Initialize a local store:

```bash
sandnote init
```

Create a workspace, an entry, and a thread:

```bash
sandnote workspace create --id ws_auth --name task/auth
sandnote entry create --id en_auth --subject "auth anchor" --meaning "resume auth work here"
sandnote thread create --id th_auth --question "How should auth work continue?" --workspace ws_auth
```

Attach the entry, focus the workspace, and resume:

```bash
sandnote entry attach en_auth --thread th_auth
sandnote workspace focus ws_auth th_auth
sandnote workspace use ws_auth
sandnote resume
```

Leave behind a better stopping point:

```bash
sandnote thread checkpoint th_auth \
  --belief "auth flow is working" \
  --open-edge "promote durable auth understanding" \
  --next-lean "promote auth topic" \
  --reentry-anchor en_auth
```

Promote durable understanding into a topic surface:

```bash
sandnote topic create --id tp_auth --name auth
sandnote topic promote tp_auth --thread th_auth --include-supporting
```

Use the REPL as a stateful working console:

```bash
sandnote repl
```

Inside the REPL:

```text
workspace use ws_auth
resume
inspect
checkpoint belief=auth-flow-is-working edge=promote-durable-auth-understanding lean=promote-auth-topic anchor=en_auth
transition dormant
exit
```

## Checkpoint Quality

Sandnote v0 centers on checkpoint quality.

A good checkpoint should minimally leave:

- a current stance
- an open edge
- a likely next lean
- a re-entry anchor

For `live` threads, Sandnote currently enforces the minimum continuity contract:

- `open_edge` must be clear enough to leave a real continuation point
- `reentry_anchor` must be present so future work has a low-cost way back in

## Thread Lifecycle

Threads carry vitality states:

- `live`
- `dormant`
- `settled`

Promotion is separate from vitality:

- vitality answers whether a thread is still alive as a line of thought
- promotion answers whether some understanding is worth preserving as a durable topic-level re-entry point

## Status

Sandnote is now in the **v0 preview hardening** stage.

The main remaining work is:

- harden end-to-end notebook workflows
- tighten the CLI contract and help/documentation
- prepare the first preview release boundary

## Preview Scope

The first v0 preview is intended to cover:

- filesystem-backed notebook state
- canonical `entry`, `thread`, `workspace`, and `topic` commands
- top-level `resume`
- persisted REPL session state
- frontier-based active work selection
- checkpoint quality enforcement for live threads

The preview is not trying to ship:

- LLM-assisted workflows
- full PKM/editor features
- synchronization or multi-user coordination
- a stable long-term storage schema beyond the current local object model

## Non-Goals

Sandnote is not trying to become:

- a memory store
- a local database with notebook branding
- a full PKM suite
- a document editor first
- an AI wrapper that auto-generates understanding

It should stay focused on resumability, checkpoint quality, and thread-first work.
