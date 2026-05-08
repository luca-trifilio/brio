---
id: brio-v2-architecture
description: Implement brio v2: format-agnostic canonical collection model with plugin interface (CollectionLoader), collection registry with fuzzy picker, jq response filtering, diagnostics pane, and theme plugin system
status: implemented
created: 2026-05-08T00:00:00Z
---
# Task: brio v2 Architecture Implementation

Implement brio v2 as described in docs/prd-v2-architecture.md and docs/bubbletea-architecture.md.

Key deliverables:
1. `canonical` package — format-agnostic Collection, Request, Environment, AuthBlock, etc. types
2. `CollectionLoader` plugin interface under `plugins/`
3. Bruno plugin wrapping existing parser/model
4. Postman plugin (collection.json)
5. Collection registry with fuzzy picker overlay (triggered by `gc`)
6. jq filter bar in response pane (toggled by `|`)
7. Diagnostics pane (toggled by `gd`)
8. Theme plugin interface under `themes/`
9. Config migration: `collections []string` → `[]CollectionEntry{Path, Format}`
