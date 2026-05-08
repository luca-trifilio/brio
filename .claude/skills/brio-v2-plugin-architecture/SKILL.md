---
name: brio-v2-plugin-architecture
description: Use when adding a new collection format loader, extending the canonical model, debugging the plugin registry, working with the collection manager modal, or understanding how the v2 architecture connects canonical types to the TUI.
---

# brio v2 Plugin Architecture

## Overview

brio v2 uses a strangler-fig migration: a format-agnostic `canonical` model lives alongside the original `internal/model` (Bruno-internal), connected via a `CollectionLoader` plugin interface. The TUI and interp/httpx layers consume only canonical types.

## Key packages

| Package | Role |
|---|---|
| `internal/canonical/` | Format-agnostic types: Collection, Folder, Request, Environment, Var, AuthBlock, Diagnostic |
| `internal/plugins/` | `CollectionLoader` interface + thread-safe `Registry` + `AutodetectLoader` opt-in interface |
| `internal/plugins/bruno/` | Bruno loader — wraps `model.LoadCollection`, adapts to canonical; implements `AutodetectLoader` |
| `internal/config/` | `CollectionEntry{Path, Format}`, `AddCollection`, `RemoveCollection`, `Entries()` |
| `internal/tui/panes/collections.go` | Multi-step collection manager modal |

## CollectionLoader interface

```go
type CollectionLoader interface {
    Name() string
    Detect(root string) bool
    Load(root string) (*canonical.Collection, []canonical.Diagnostic, error)
}
```

`Detect` is used when `format` is empty in config — the registry probes all loaders and returns the first match.

## AutodetectLoader interface (opt-in)

Separate from `CollectionLoader` — checked via type assertion so existing loaders aren't broken:

```go
type AutodetectLoader interface {
    Autodetect() []string  // returns absolute paths where Detect() would return true
}
```

Bruno implements this by scanning `brunoprefs.CollectionPaths()` + CWD + immediate children.
The import modal checks `_, ok := loader.(plugins.AutodetectLoader)` to show/hide the Autodetect option.

## Adding a new format loader

1. Create `internal/plugins/<format>/loader.go` implementing `CollectionLoader`
2. Self-register via `init()`:
   ```go
   func init() { plugins.Register(New()) }
   ```
3. Blank-import in `internal/cli/root.go`:
   ```go
   _ "github.com/luca-trifilio/brio/internal/plugins/<format>"
   ```
4. Write adapter translating the format's native types → canonical types
5. Test: verify `Detect` on a real collection root and `Load` produces non-empty canonical output
6. Optionally implement `AutodetectLoader` if the format has a known preferences/discovery mechanism

## Diagnostics flow

- `Loader.Load()` returns `[]canonical.Diagnostic` for per-file parse errors (non-fatal)
- `tui.LoadCollections()` aggregates diagnostics from all loaders into `m.diagnostics`
- Status bar shows `⚠ N` badge when `len(m.diagnostics) > 0`
- `gd` keychord opens the diagnostics overlay pane

## Config shape

New `[[collections]]` TOML tables (as of v2.1 import feature):

```toml
[[collections]]
path = "/path/to/api"
format = "bruno"
```

Legacy flat strings still load (backward-compat pre-pass in `Load()`), but `Save()` always
writes the table form. `Config.Collections` is `[]CollectionEntry{Path, Format}`.

Key config methods:
- `AddCollection(path, format string) bool` — dedupes by expanded abs path, returns false if already present
- `RemoveCollection(path string) bool` — removes by expanded abs path
- `Entries()` — trivial pass-through (was a legacy-bridge, now direct)

## Collection manager modal (CollectionsModel)

Multi-step modal in `internal/tui/panes/collections.go`:

| Step | Name | Description |
|------|------|-------------|
| stepList | List | Existing collections + "Add collection". Skipped on Open() when entries empty. |
| stepPlugin | Plugin pick | Skipped when only 1 plugin registered |
| stepPath | Path input | Text input + optional Autodetect row (Tab to toggle) |
| stepCandList | Candidate select | Multi-select list of detected collections |
| stepConfirm | Confirm | Shows paths + format, Save/Cancel buttons |

Signal enum: `CollMgrContinue` / `CollMgrSaved` / `CollMgrCanceled`

- `Open(entries)` with empty entries calls `beginAdd()` directly (skips the empty list screen)
- On `CollMgrSaved`: caller reads `Entries()` and calls `config.Save`

Wired in `app.go`:
- `I` keybinding opens modal from any pane (normal mode)
- After `LoadCollections`, if `len(collections) == 0` → `OpenCollMgrIfEmpty()` auto-opens modal
- CLI no longer has Bruno prefs fallback — autodetect is user-initiated from the modal only

## Bubble Tea textinput gotcha

When accumulating typed characters via `HandleKey(key string)` rather than `Update(tea.Msg)`, keep a **plain string buffer** as the source of truth. Do NOT read from `textinput.Value()` after calling `SetValue()` manually — it can silently drop characters in real terminal environments.

```go
// Correct pattern:
filterBuf += key              // authoritative accumulator
filterInput.SetValue(filterBuf) // rendering only

// On submit:
query := strings.TrimSpace(filterBuf) // read from buf, not filterInput.Value()
```

## Gotcha: TreeModel.clamp() with zero rows

When collections is empty, `clamp()` must short-circuit before cursor/offset arithmetic.
Without the guard, `t.offset = t.Cursor` can set a negative offset → index-out-of-range panic on first `View()`.

```go
func (t *TreeModel) clamp() {
    if len(t.rows) == 0 {
        t.Cursor = 0
        t.offset = 0
        return
    }
    // ... rest of clamp
}
```

Apply the same pattern to any Bubble Tea list/tree model that can have zero items.

## Deferred features (plugin hooks exist, implementations follow)

- **Postman loader** (v2.2) — `CollectionLoader` interface is the hook; add `internal/plugins/postman/`; implement `AutodetectLoader` if Postman has a known prefs location
- **Theme system** (v2.2) — `theme = ""` placeholder in config; no registry yet
