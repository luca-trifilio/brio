# ADR-0001: Use ScriptBlock field on canonical.Request instead of Extra map

---
id: ADR-0001
title: Use ScriptBlock field on canonical.Request instead of Extra map
status: Accepted
date: 2026-05-08
---

## Context

The canonical model must carry pre/post script text so the interp layer
can execute them without knowing the source format. The Bruno loader
has access to the raw BruDoc AST at load time and can extract script
text from it. Two approaches were considered for how canonical types
should expose this data.

## Decision

Add `Scripts ScriptBlock{Pre, Post string}` to `canonical.Request`.
The Bruno loader extracts pre/post script text from the BruDoc AST
during the model→canonical translation, storing plain text.
The interp layer reads `Request.Scripts.Pre/Post` directly.
`Extra map[string]any` is reserved for opaque vendor metadata that
no internal package needs to read.

## Alternatives Considered

- **Store BruDoc AST handle in `Extra["bruDoc"]`**: rejected because
  it routes a format-specific AST reference through a nominally
  format-agnostic type, requires type-asserts in the interp hot path,
  and makes canonical types non-portable.
- **Keep interp/script.go reading from `*model.Request` directly**:
  rejected because it prevents other loaders from providing scripts
  and defeats the purpose of the canonical abstraction.

## Consequences

**Positive:**
- canonical.Request is genuinely format-agnostic: any loader can
  populate Scripts without leaking its internal representation.
- No type-asserts in interp hot path.
- Future Postman loader populates the same field from its own AST.

**Negative / Trade-offs:**
- Bruno loader must extract all script text upfront at load time
  (not lazily). For large collections this is acceptable; scripts
  are small text blocks.
