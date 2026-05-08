# ADR-0002: Build and test Bruno adapter before wiring TUI to canonical types

---
id: ADR-0002
title: Build and test Bruno adapter before wiring TUI to canonical types
status: Accepted
date: 2026-05-08
---

## Context

The v2 migration replaces *model.Collection with *canonical.Collection
throughout app.go (1069 lines) and all panes. app.go has a test suite
that must remain green. Doing the TUI rewire and the model translation
in the same step risks a large, hard-to-bisect diff.

## Decision

Build the Bruno model→canonical adapter (internal/plugins/bruno/adapter.go)
and validate it against the existing model golden-output tests before
touching app.go or any pane. Only once the adapter is green do we
flip the TUI field types. The TUI rewire then becomes a pure type-swap
with no logic changes.

## Alternatives Considered

- **Rewire TUI and adapter in parallel**: rejected because a regression
  in either layer is harder to isolate; the combined diff is large and
  review-unfriendly.
- **Big-bang rewrite of model + TUI in one PR**: rejected because it
  cannot be incrementally validated and breaks bisect if tests fail.

## Consequences

**Positive:**
- Existing tests act as a safety net before any TUI change lands.
- TUI diff is a mechanical type-swap, easy to review.
- Adapter can be merged and used independently.

**Negative / Trade-offs:**
- Two commits instead of one for what could be a single feature.
- Adapter must faithfully reproduce all model fields or tests will
  catch mismatches (acceptable — that is the point).
