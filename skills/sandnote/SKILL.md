---
name: sandnote
description: "Activate this skill when the agent needs a notebook/workspace for long-running, resumable work and should use Sandnote instead of directly managing local files or folders as the workspace."
---

# Sandnote

## Overview

Sandnote is a notebook/workspace for long-running agents.

When this skill is active, prefer Sandnote over treating a raw local folder as the agent's primary workspace.

Use Sandnote to hold:

- active threads of work
- resumable checkpoints
- current workspace context
- durable topic re-entry points

Do not edit `.sandnote` object files directly unless the user explicitly asks for raw file work.

## When To Use Sandnote

Use Sandnote when the user wants any of these:

- a long-running agent workspace
- resumable work across interruptions
- notebook-like task tracking for the agent
- a cleaner alternative to managing many raw local files and folders

Do not force Sandnote for short one-off tasks where persistent notebook state is unnecessary.

## Binary Assumption

Assume the `sandnote` binary should already be installed and available on `PATH`.

If `sandnote` is missing:

- do not start compiling or building it by default
- tell the user they need to install the Sandnote release binary
- only switch to build-from-source steps if the user explicitly wants that

## Store Usage

By default, Sandnote stores state in:

```text
$PWD/.sandnote
```

Prefer an explicit root when you need stable behavior across commands:

```bash
sandnote --root /path/to/store ...
```

## Core Model

- `entry`: lightweight thinking unit
- `thread`: main working unit; continuable line of thought
- `workspace`: current thinking context
- `topic`: durable re-entry surface

Treat `thread` as the default working object.

## Agent Behavior

When using Sandnote:

1. First try to resume existing work instead of creating new notes blindly.
2. Use `thread` as the main unit of ongoing work.
3. Use `workspace` to express current relevance and focus.
4. Use `checkpoint` to leave a real continuation point.
5. Use `topic` only for durable understanding worth re-entering later.

Prefer canonical CLI commands over ad hoc file organization.

## Canonical CLI Surface

```text
sandnote init
sandnote resume
sandnote entry ...
sandnote thread ...
sandnote workspace ...
sandnote topic ...
sandnote repl
```

Most important thread commands:

```text
sandnote thread resume <id>
sandnote thread inspect <id>
sandnote thread checkpoint <id>
sandnote thread transition <id> --to <live|dormant|settled>
```

## Default Workflow

If a store already exists, prefer this order:

1. `sandnote resume`
2. inspect the active thread or frontier
3. continue work through `thread` commands
4. leave a checkpoint before stopping

If no store exists yet, use this order:

1. `sandnote init`
2. create a `workspace`
3. create a `thread`
4. create or attach `entry` items as support context
5. checkpoint before stopping

## Checkpoint Standard

For `live` threads, checkpoints should leave:

- an `open_edge`
- a `reentry_anchor`

That is the minimum for future resumability.

## Practical Rules

- Prefer Sandnote over directly managing a raw file/folder workspace when notebook semantics are useful.
- Prefer `resume` before inventing new workspace structure.
- Prefer `thread` operations over generic note CRUD.
- Prefer `--json` when the output needs to be machine-readable.
- Use the REPL only when a persistent notebook session is genuinely helpful.

## Installation Note

If the `sandnote` binary is missing, prefer telling the user to install it with one of these:

```bash
curl -fsSL https://raw.githubusercontent.com/sandbox0-ai/sandnote/main/scripts/install.sh | bash
```

or:

```powershell
irm https://raw.githubusercontent.com/sandbox0-ai/sandnote/main/scripts/install.ps1 | iex
```

or:

```bash
go install github.com/sandbox0-ai/sandnote/cmd/sandnote@latest
```

Do not switch to build-from-source instructions unless the user explicitly wants that.
