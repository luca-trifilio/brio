---
name: bubbletea-scroll-pane
description: Patterns for stateful scrollable content panes in Bubble Tea + Lip Gloss. Use when adding vim-scroll to a read-only pane, fixing ANSI line-wrap truncation, adding separator rows to a tree widget, classifying environments by safety tier, or detecting content changes in View() without explicit SetX() calls.
---

# Bubbletea Scroll Pane Patterns

Companion files — read the relevant one when the topic comes up:

| Topic | File |
|---|---|
| ANSI-safe wrapping, `ansi.Hardwrap`, two-pass rebuild to avoid `pad()` truncation | `./wrapping.md` |
| Tree separator rows between groups, `NodeSeparator`, `skipSeparator()` navigation | `./tree-separators.md` |
| Env safety tiers, `ClassifyEnv`, blocked HTTP methods, tier-aware sort | `./env-tiers.md` |
| Smart `View()` change detection via `lastReq`/`lastEnv` without explicit `SetX()` | `./change-detection.md` |
| Scrollbar `│` glued to text (`pad` vs `truncate`), hide empty folders in filtered tree | `./scrollbar-alignment.md` |

## When to reach for each file

- Getting `…` truncation on wrapped lines → `./wrapping.md`
- Adding blank space / visual grouping between tree sections → `./tree-separators.md`
- Colouring envs by danger level or hiding mutating methods in prod → `./env-tiers.md`
- Pane content depends on external state (selected item + active env) → `./change-detection.md`
- Scrollbar character appears right after text instead of at the right edge → `./scrollbar-alignment.md`
- Empty folder nodes visible in tree after a method filter is applied → `./scrollbar-alignment.md`
