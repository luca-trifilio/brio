# ADR-0005: Migrate Collections Config Schema to TOML Tables

---
id: ADR-0005
title: Migrate Collections Config Schema to TOML Tables
status: Accepted
date: 2026-05-08
---

## Context

The original `config.toml` schema stores collection paths as a flat string array:

    collections = ["/path/to/api", "/path/to/other"]

The import feature requires storing per-collection metadata alongside the path
(at minimum: the detected format name, e.g. "bruno"). The flat string schema
has no slot for this. Since the import feature already requires mutating
config.toml, this is the least-disruptive moment to migrate the on-disk schema
to a richer shape.

## Decision

Migrate the on-disk collections schema to TOML array-of-tables:

    [[collections]]
    path = "/path/to/api"
    format = "bruno"

    [[collections]]
    path = "/path/to/other"
    format = "bruno"

The in-memory `CollectionEntry{Path, Format}` struct already exists.
`Config.Collections` changes type from `[]string` to `[]CollectionEntry`.
`config.Load()` reads legacy flat strings via a backward-compat pre-pass and
promotes them to `CollectionEntry{Path: s, Format: ""}` (empty format = auto-detect).
`config.Save()` always writes the new table shape.
`Config.Entries()` becomes a trivial pass-through.

## Alternatives Considered

- **Keep flat strings, store format elsewhere**: Would require a parallel
  `[collection_formats]` map keyed by path — fragile and hard to keep in sync.
  Rejected.
- **Defer migration to a later release**: Would require a second breaking
  config change after the import feature ships. Users would migrate twice.
  Rejected.
- **Use a SQLite database**: Overkill for a single-user local TUI tool with
  O(10s) of collections. Adds a CGO dependency that breaks the static binary
  constraint (CGO_ENABLED=0). Rejected.

## Consequences

**Positive:**
- Per-collection metadata (format, and future fields) is natively representable.
- `Entries()` simplifies to a pass-through; no more string-to-struct bridging.
- One migration event for users instead of two.

**Negative / Trade-offs:**
- Breaking on-disk change: users with hand-written `collections = [...]` must
  let brio rewrite their config on first import (comments are lost on Save).
- The backward-compat load path must be tested carefully to avoid silent data loss.
