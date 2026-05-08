# ADR-0004: AutodetectLoader as an Opt-In Interface

---
id: ADR-0004
title: AutodetectLoader as an Opt-In Interface
status: Accepted
date: 2026-05-08
---

## Context

The collection import modal needs an "Autodetect" option that scans known
locations (Bruno preferences paths, CWD) and returns candidate collection
roots without the user typing a path. Only some plugins can support this
(Bruno can scan preferences.json; a future Postman plugin may not have an
equivalent). Adding Autodetect() directly to the CollectionLoader interface
would require every existing and future plugin to implement it, breaking the
Bruno loader and any third-party plugins.

## Decision

Define a separate AutodetectLoader interface:

    type AutodetectLoader interface {
        Autodetect() []string
    }

Plugins that support autodetection implement it optionally. The import modal
checks for it via type assertion:

    if ad, ok := loader.(plugins.AutodetectLoader); ok {
        paths = ad.Autodetect()
    }

The core CollectionLoader interface (Name, Detect, Load) is unchanged.

## Alternatives Considered

- **Add Autodetect() to CollectionLoader**: Would require all existing loaders
  to implement a no-op stub. Breaks the interface contract for plugins that
  genuinely cannot autodetect. Rejected.
- **Pass autodetect as a function at registration time**: More flexible but
  adds registration complexity with no clear benefit over a simple interface.
  Rejected.

## Consequences

**Positive:**
- CollectionLoader interface remains stable; existing plugins unaffected.
- Autodetect is genuinely optional — the UI can show/hide the option based on
  capability without special-casing plugin names.
- Pattern is idiomatic Go (small interfaces, type assertions for opt-in behaviour).

**Negative / Trade-offs:**
- Two interfaces to document and discover instead of one.
- Future plugin authors must know to check for AutodetectLoader separately —
  easy to miss if they only read CollectionLoader.
