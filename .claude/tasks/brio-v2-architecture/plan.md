# Plan: brio v2 Architecture

## Goal
Decouple brio from the Bruno-only `.bru` format by introducing a `canonical` model and a `CollectionLoader` plugin interface, with Bruno loader, fuzzy collection picker, jq response filter, and diagnostics pane. All `make check` gates pass; existing Bruno workflows unchanged for end users.

## Approach
Strangler-fig migration: introduce `internal/canonical` alongside existing `internal/model`, build and test the Bruno model→canonical adapter first (safety net), then wire root TUI/CLI to canonical types, then add picker, jq, diagnostics as additive features. Config schema extended in-memory only (no config file rewrite).

Resolved decisions:
- **Adapter-first**: Bruno model→canonical adapter tested against existing golden output before `app.go` is touched. TUI rewire becomes a type-swap.
- **No `Extra` in hot paths**: `canonical.Request` carries `Scripts ScriptBlock{Pre, Post string}` extracted at load time. `Extra map[string]any` reserved for opaque vendor metadata only.
- **Config**: `[]string` parsed to `[]CollectionEntry{Path, Format}` in memory; no config file rewrite in this iteration.
- **Fuzzy picker UX**: match on `canonical.Collection.Name`, path as secondary hint, `filepath.Base` fallback.
- **Diagnostics badge**: status line shows `⚠ N` when diagnostics exist; `gd` toggles overlay. No badge = no pane noise.
- **Format detection**: directory probing (`bruno.json` / `*.bru` presence) with explicit `format=` override in config.
- **Postman loader**: deferred to v2.1 (plugin interface ships in v2, implementation follows).
- **Theme system**: deferred to v2.2 (`theme = ""` placeholder in config only).
- **New dep**: only `gojq` added in this iteration.

## Steps

### 1. Add `internal/canonical` package
- **Files**: `internal/canonical/{collection,request,environment,auth,vars,diagnostic}.go`
- **Action**: Define `Collection`, `Folder`, `Request` (with `Scripts ScriptBlock{Pre, Post string}`), `Environment`, `Var`, `AuthBlock` (Bearer/Basic/APIKey/AwsV4), `Body`, `Settings`, `Diagnostic{Severity,Path,Line,Msg}`. Each type may carry `Extra map[string]any` for opaque vendor metadata. No AST references.
- **Verify**: `go build ./...`; godoc comments end with period (godot).

### 2. Define `CollectionLoader` plugin interface
- **Files**: `internal/plugins/loader.go`, `internal/plugins/registry.go`
- **Action**: Interface `CollectionLoader { Name() string; Detect(root string) bool; Load(root string) (*canonical.Collection, []canonical.Diagnostic, error) }`. Registry with `Register(loader)`, `Resolve(format, root)`, `DetectAll(root)`.
- **Verify**: Unit test registry registration/lookup.

### 3. Bruno loader + adapter (safety net before TUI rewire)
- **Files**: `internal/plugins/bruno/loader.go`, `internal/plugins/bruno/adapter.go`, `bruno_test.go`
- **Action**: Wrap `model.LoadCollection`. Adapter translates `*model.Collection` → `*canonical.Collection`. Bruno loader extracts pre/post script text from `BruDoc` via `FindBlock` at load time, placing text in `canonical.Request.Scripts`. `Detect` checks for `bruno.json` or any `*.bru` files. Tests reproduce existing `loader_test.go` golden output via adapter output.
- **Verify**: All existing model tests pass; adapter test confirms field-by-field equivalence.

### 4. Config schema extension (in-memory only)
- **Files**: `internal/config/{config,load}.go`, `config_test.go`
- **Action**: Introduce `CollectionEntry{Path, Format string}`. At load time, convert legacy `Collections []string` → `[]CollectionEntry` in memory (no file write). Add `ActiveCollection string`, `Theme string` (placeholder, unused). No `.bak`, no config rewrite.
- **Verify**: Round-trip test: legacy TOML with `[]string` loads correctly as `[]CollectionEntry`; `make check`.

### 5. Wire CLI + root TUI to plugin registry
- **Files**: `internal/cli/root.go`, `internal/tui/app.go`
- **Action**: Replace `model.LoadCollection` with `plugins.Resolve(entry.Format, entry.Path).Load(...)`. `Model.collections` becomes `[]*canonical.Collection`. Update all panes consuming `*model.Collection` (tree, env, vars, request) to use canonical types. Aggregate `[]canonical.Diagnostic` from all loader calls.
- **Verify**: `internal/tui/app_test.go` still green; manual smoke against existing Bruno collection.

### 6. Update interp + httpx for canonical types
- **Files**: `internal/interp/{vars,auth,script}.go`, `internal/httpx/executor.go`
- **Action**: `VarScope` consumes `canonical.Var`; auth chain walks canonical folder chain. `script.go` reads `Request.Scripts.Pre/Post` (plain text, no AST). Add APIKey resolver in `httpx`.
- **Verify**: Existing interp/httpx tests adapted; `make check`.

### 7. Collection fuzzy picker overlay
- **Files**: `internal/tui/panes/picker.go`, integration in `internal/tui/app.go`
- **Action**: New pane modeled on settings.go CollEditor pattern (`Continue/Saved/Cancelled` enum). Triggered by `gc` keychord. Filters on `canonical.Collection.Name` (secondary: path, fallback: `filepath.Base`). On select, switches `Model.activeCollection`.
- **Verify**: Unit test picker filtering logic; manual `gc` toggle smoke test.

### 8. jq response filter bar
- **Files**: `internal/tui/panes/response.go`, `go.mod`
- **Action**: `go get github.com/itchyny/gojq`. Add `filterMode bool`, `textinput.Model` for query. Toggle on `|`. On Enter, run `gojq.Parse(query)` against last response JSON; render filtered output, fall back to original on error (show error in status line). `Esc` clears filter.
- **Verify**: Table-driven tests for pure helper `applyJQ(json, query) (string, error)`.

### 9. Diagnostics overlay pane + status badge
- **Files**: `internal/tui/panes/diagnostics.go`, `internal/tui/app.go` (status line render)
- **Action**: Modal overlay listing `[]canonical.Diagnostic` from loader calls. Status line shows `⚠ N` badge when N > 0. Toggle `gd`. List with severity icon, path:line, message. `Esc` closes. `Enter` opens file in `$EDITOR` (best effort).
- **Verify**: Render test with synthetic diagnostics; confirm badge appears/disappears correctly.

### 10. Docs + final polish
- **Files**: `README.md`, keymap docs, CHANGELOG entry
- **Action**: Document `gc`, `|`, `gd`, `CollectionEntry` config format, plugin interface for future loaders.
- **Verify**: `make check` clean.

## Risks
- **`app.go` TUI rewire scope creep** → adapter-first approach (Step 3) provides a tested safety net; rewire in Step 5 is purely a type-swap.
- **gojq panics on malformed JSON** → wrap filter execution in `recover()`; surface as status line error, not crash.
- **Fuzzy picker empty state** → if no collections configured, show instructional hint rather than empty list.
- **Diagnostics badge confusion** → badge only appears on load; `gd` opens overlay even when badge is zero (for discoverability), but badge is the primary signal.
- **TUI pane refactor regressions** → keep `internal/tui/app_test.go` invariant; add canonical-typed fixtures before flipping field types in Step 5.
- **Format detection ambiguity** → if both `bruno.json` and `*.bru` absent, loader returns a diagnostic rather than silently loading nothing.

## Out of Scope (this iteration)
- Postman loader (deferred — plugin interface is the hook).
- Theme system (deferred — `theme = ""` placeholder in config only).
- Auth extensibility (deferred — `AuthBlock` in canonical includes Bearer/Basic/APIKey/AwsV4 fields, but the `Authenticator` plugin interface and hook-based auth are a follow-up iteration; see `docs/roadmap.md`).
- Config file rewrite / migration on disk.
- Insomnia / OpenAPI / HAR loaders.
- Writing back to collection files (read-only invariant preserved).
- Persisting jq filters per-request.
- LSP-style live diagnostics (one-shot at load time only).
- Theme hot-reload.
- Unifying script engines.
- WebSocket / gRPC / GraphQL request types.

## Roadmap (follow-up iterations)
- **Next**: Postman collection.json loader (goja for scripts; plugin interface is the hook).
- **Later**: Theme plugin system (builtin Catppuccin variants + Tokyo Night, user TOML override).
