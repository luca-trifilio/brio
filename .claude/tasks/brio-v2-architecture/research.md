# Research: brio v2 Architecture

## Summary
brio is a single-format Bruno TUI today: a custom `.bru` parser feeds a tightly-coupled model layer (`Request.Doc *parser.BruDoc`) that leaks AST references into the interpolation and script engines. The v2 task requires a clean format-agnostic canonical model, a `CollectionLoader` plugin interface, fuzzy collection picker, jq response filtering, diagnostics pane, and a theme registry — none of which have dependencies (goja, gojq, fuzzy lib) yet present in go.mod.

## Relevant Files

### Model & Parser
- `internal/parser/parser.go`, `ast.go` — `.bru` block parser. `BruDoc{Blocks[]}`, `Block{Name,Subtype,Lines,Raw}`. Must stay internal to future Bruno plugin.
- `internal/model/collection.go`, `request.go`, `loader.go`, `types.go` — current "canonical" types: `Collection`, `Folder`, `Request`, `Environment`, `Var`, `AuthBlock`, `Body`, `Settings`. `LoadCollection(root)` walks dirs, calls `parser.ParseFile`. These become the Bruno-internal types; v2 introduces a separate `canonical` package.
- `internal/model/request.go` — `Request.Doc *parser.BruDoc` leak used by `interp/script.go` for `FindBlock` lookups. Core coupling to break.

### Interpolation & Auth
- `internal/interp/vars.go` — layered `VarScope` (collection → env → folder chain → request → runtime). Cycle detection. Must be extended for canonical `VarScope`.
- `internal/interp/auth.go` — auth inheritance chain walker using `Folder.FolderAuth`. Pattern reusable.
- `internal/interp/script.go` — regex-based pre/post-response script runner (no JS engine). Bruno plugin keeps this; Postman plugin will need goja.

### HTTP Executor
- `internal/httpx/executor.go`, `types.go`, `sigv4.go` — clean `ResolvedRequest → Response` boundary, already format-agnostic. Needs bearer/basic/apikey auth extension.

### TUI
- `internal/tui/app.go` (1069 lines) — root Bubble Tea model. Panes: tree, env, vars, history, response, request, help, settings. `Model.collections []*model.Collection`. Mouse hit-testing via `paneGeometry`.
- `internal/tui/panes/settings.go` (367 lines) — CollEditor sub-mode pattern with `Continue/Saved/Cancelled` signal enum. **Prior art for fuzzy picker overlay.**
- `internal/tui/panes/response.go` (852 lines) — vim motions, leap, visual selection, chroma syntax highlight. jq filter bar wires here.
- `internal/tui/layout.go` — fixed 4-pane layout (sidebar: tree+env, right: request+response). Diagnostics pane needs layout change or overlay.

### Config
- `internal/config/config.go`, `load.go`, `save.go`, `template.go` — TOML config: `Collections []string`, `Hooks []Hook`. XDG path. No theme/plugins fields. Migration needed: `[]string` → `[]CollectionEntry{Path, Format}`.

### Theme
- `internal/theme/theme.go` (197 lines) — hardcoded Catppuccin Macchiato palette + semantic styles + env-tier classifier. No theme registry/loading. Stays default but must become one option among many.

### CLI
- `internal/cli/root.go` — cobra entrypoint, calls `model.LoadCollection` directly. Must be updated to use plugin registry.

### Hooks
- `internal/hooks/` — `tea.Cmd`-based hook runner with stdout/file output formats. Pattern reusable for plugin loading hooks.

## Existing Patterns
- **Bubble Tea message flow**: `executeMsg`, `hooks.DoneMsg`, `editorDoneMsg` — cmd-driven async ops.
- **Modal sub-modes**: boolean flag + embedded model + absorb-keys pattern (settings pane CollEditor).
- **Best-effort loaders**: parse failures don't abort collection load — diagnostic errors go to status line today, v2 moves them to diagnostics pane.
- **Auth inheritance**: folder chain walk already abstracts auth from request layer.
- **Read-only invariant**: no `.bru` writes anywhere. Constraint extends to v2.

## Current Model Types (for canonical equivalents)
- `model.Collection{Name, Description, Root string, Folders []*Folder, Requests []*Request, Vars []Var, Environments []Environment, Auth AuthBlock, Settings Settings}`
- `model.Request{Name, Path, Method, URL, Headers, Params, Body, Auth AuthBlock, Vars []Var, Doc *parser.BruDoc}` — `Doc` field is the coupling to remove
- `model.AuthBlock{Mode, Bearer, Basic, AwsV4}` — extend with `APIKey`
- `model.Environment{Name, Vars []Var}`
- `model.Var{Name, Value, Local bool}`

## TUI Architecture
- Root `Model` in `app.go` owns all panes as fields
- Focus tracked via `focusedPane` enum in root
- Overlay pattern: boolean `showModal` + early-return key routing
- `tea.WindowSizeMsg` broadcast to all panes
- Status line rendered by root `View()`
- Mode enum (`normal/insert/command`) in root model

## Config Structure
```toml
[settings]
collections = ["/path/to/collection"]  # currently []string
active_environment = "prod"
# Missing: active_collection, theme
```

Migration: on first load, plain string entries silently upgraded to `CollectionEntry{Path: s}`.

## Dependencies
- **Present**: BurntSushi/toml, charmbracelet/{bubbles,bubbletea,lipgloss}, alecthomas/chroma, atotto/clipboard, aws-sdk-go-v2, spf13/cobra
- **NOT present** (must add): `goja` (JS runtime for Postman scripts), `gojq` (pure-Go jq), fuzzy lib (e.g. `sahilm/fuzzy`)

## Constraints
- `make check` (fmt+vet+lint+test) gates all commits. golangci-lint: godot (exported comments must end with period), errcheck, noctx.
- goimports with local prefix `github.com/luca-trifilio/brio`.
- Table-driven tests; `testing.Short()` for slow paths.
- Existing tests in: `internal/parser/`, `internal/interp/`, `internal/httpx/`, `internal/model/loader_test.go`, `internal/config/`, `internal/tui/app_test.go`.
- Read-only invariant: v2 must not write to `.bru` or any collection format files.
- Go 1.26.2 (very recent — no compatibility concerns with generics etc.).
- `CGO_ENABLED=0`: static binaries, no C deps — `goja` and `gojq` are pure Go, safe.
- Binary size: `goja` adds ~5MB (acceptable per PRD).

## Open Questions
1. **CollectionLoader detection**: file-extension dispatch, directory probing (presence of `bruno.json` vs `*.postman_collection.json`), or explicit `[[collection]] format="bruno"` in config?
2. **CanonicalRequest.Extra**: keep `Extra map[string]any` for loader-specific data (scripts, vendor metadata)?
3. **jq filter scope**: per-response only, or persistable per-request? UX: always-on textinput vs toggle with `|`?
4. **Diagnostics pane layout**: 5th pane (requires layout.go change) or modal overlay (simpler, uses existing overlay pattern)?
5. **Theme plugin**: built-in compiled themes (Catppuccin variants, Tokyo Night) + user TOML override, or only user-defined? Hot-reload?
6. **Script engine**: Bruno plugin keeps regex-based runner; Postman plugin adds goja. Or unify everything on goja (bigger but consistent)?
7. **Migration scope**: keep `model.*` types as Bruno-internal and introduce parallel `canonical.*`, or rename existing model to canonical and add Bruno adapter shim?
