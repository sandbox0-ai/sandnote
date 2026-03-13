# sandnote

**An Obsidian-like, CLI-first notebook engine backed by the filesystem.**

Sandnote is a CLI-first, filesystem-backed, headless notebook engine for agents.

It is not agent memory. It is the agent's notebook.

Its core purpose is to preserve the continuity of thought.

## Product Promise

**Come back to unfinished thought without starting over.**

For v0, the more operational framing is:

**Sandnote should help you stop without losing your place.**

## What Sandnote Is

Sandnote is an external thinking surface built on top of the filesystem.

It should be:

- resumability-first, not storage-first
- thread-first in active work, not note-first
- context-aware, not container-first
- focused on re-entry value, not information volume

## What Sandnote Is Not

Sandnote should not be defined as:

- a memory store
- a local database with a notebook veneer
- a full PKM suite
- a document editor first
- an AI wrapper that auto-generates understanding

## Core Model

A useful product model is:

- `entry`: a lightweight thinking unit
- `thread`: a continuable line of thought
- `workspace`: the current thinking surface that gives context
- `topic surface`: a durable re-entry surface for future work

These layers should stay distinct.

## Why Filesystem-Backed

Sandnote is built on the filesystem because the filesystem is the durable substrate:

- files are the source of truth
- note content remains human-readable
- snapshots and restores come from the underlying volume
- derived indexes and metadata can be rebuilt

Direct file editing should remain possible, but raw file editing alone is not enough to provide notebook semantics.

## V0 Center Of Gravity

If v0 must focus on one thing, it should focus on checkpoint quality.

The key question is not whether a session produced more text, but whether it left behind a better edge for future continuation.

A good checkpoint should minimally leave:

- a current stance
- an open edge
- a likely next lean
- a re-entry anchor

## Lifecycle

Threads should have vitality states such as:

- `live`
- `dormant`
- `settled`

Promotion should remain a separate dimension from vitality:

- vitality answers whether a thread is still alive as a line of thought
- promotion answers whether some understanding is worth preserving as a durable topic-level re-entry point

## Product Discipline

Sandnote should avoid drifting into:

- content management first
- structural completeness first
- long-lived knowledge accumulation first
- premature AI automation first

Instead, it should optimize for making interrupted thought easier to resume.

## Status

Sandnote is in the product-definition stage.

The current priority is to turn the product thesis into a concrete v0 semantics and CLI surface, without losing the central design goal: preserving the continuity of thought.
