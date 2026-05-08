# Progress: brio v2 Architecture Implementation

## Status
current_step: 10
started: 2026-05-08T00:00:00Z
last_update: 2026-05-08T15:30:00Z
outcome: completed

## Steps
- [x] Step 1: Add `internal/canonical` package
- [x] Step 2: Define `CollectionLoader` plugin interface
- [x] Step 3: Bruno loader + adapter (safety net before TUI rewire)
- [x] Step 4: Config schema extension (in-memory only)
- [x] Step 5: Wire CLI + root TUI to plugin registry
- [x] Step 6: Update interp + httpx for canonical types
- [x] Step 7: Collection fuzzy picker overlay
- [x] Step 8: jq response filter bar
- [x] Step 9: Diagnostics overlay pane + status badge
- [x] Step 10: Docs + final polish

## Log

### Step 1: Add `internal/canonical` package
**Status**: completed
**Timestamp**: 2026-05-08T11:00:00Z
Created `internal/canonical/` with files: collection.go (Collection + Folder + AllRequests), request.go (Request, Header, Param, Body, ScriptBlock, HTTPMethod), auth.go (AuthMode, AuthBlock, AuthBearerCfg, AuthBasicCfg, AuthAPIKeyCfg, AuthAWSv4Cfg), vars.go (Var with Local), environment.go (Environment), settings.go (Settings), diagnostic.go (Severity, Diagnostic). All types carry `Extra map[string]any` per plan. Godot-compliant doc comments. `go build ./...` clean.

### Step 2: Define `CollectionLoader` plugin interface
**Status**: completed
**Timestamp**: 2026-05-08T11:15:00Z
Created `internal/plugins/loader.go` with `CollectionLoader` interface (Name, Detect, Load) and `internal/plugins/registry.go` with thread-safe `Registry` (Register, Resolve, DetectAll) plus package-level `Default()`, `Register()`, `Resolve()` helpers.

### Step 3: Bruno loader + adapter (safety net before TUI rewire)
**Status**: completed
**Timestamp**: 2026-05-08T11:30:00Z
Created `internal/plugins/bruno/`: loader.go (CollectionLoader implementation; self-registers via init()), adapter.go (model→canonical translation), loader_test.go (Detect + Load coverage on testdata).

### Step 4: Config schema extension (in-memory only)
**Status**: completed
**Timestamp**: 2026-05-08T11:45:00Z
Added `ActiveCollection`, `Theme` fields and `CollectionEntry{Path, Format}` with `(*Config).Entries()` to `internal/config/config.go`. No on-disk migration. Test in entries_test.go.

### Step 5: Wire CLI + root TUI to plugin registry
**Status**: completed
**Timestamp**: 2026-05-08T15:00:00Z
- Extended `canonical.Collection`/`canonical.Folder` with a `Scripts ScriptBlock` field so pre/post script source survives the model→canonical translation.
- Added `canonical.Collection.EnvByName`, `EnvNames`, `DisplayName` helpers.
- Bruno adapter now extracts collection-level and folder-level pre/post scripts via `BruDoc.FindBlock`.
- `internal/cli/root.go` blank-imports `internal/plugins/bruno` and uses `tui.LoadCollections([]config.CollectionEntry)`, which resolves loaders via `plugins.Resolve(format, root)` and aggregates `[]canonical.Diagnostic`.
- `internal/tui/app.go`: `Model.collections` is now `[]*canonical.Collection`; `activeCollection`, `collectionFor`, `envFor`, `activeRequestAndScope` updated; `m.activeEnvs` keyed on `c.Root` (since canonical drops `Path`); `editorDoneMsg` reload path uses the plugin registry.
- `internal/tui/actions.go` rebuilt against canonical: `resolveRequest(c, env, req, runtime)` consumes canonical types; `executeMsg`/`runRequestCmd`/history conversion all updated.
- Panes (`tree.go`, `env.go`, `request.go`) migrated to canonical types and `Folder.Path` / `EnvByName` accessors. Tree's `Path` for collection rows now stores `c.Root`.
- `internal/tui/app_test.go` rewritten to construct canonical fixtures.
- `make check` clean.

### Step 6: Update interp + httpx for canonical types
**Status**: completed
**Timestamp**: 2026-05-08T15:00:00Z
- `internal/interp/vars.go`: `NamedLayer.Vars`/`Push` use `[]canonical.Var`.
- `internal/interp/auth.go`: `ResolveAuth(c, req)` and `BuildScope(c, env, req, runtime)` accept canonical types; `FolderChainFor` walks `RootFolder.Folders` instead of `Root.Folders`.
- `internal/interp/script.go`: `CollectPreRequestVars` reads `Scripts.Pre` from canonical Collection / Folder / Request rather than `BruDoc.FindBlock`.
- `internal/tui/actions.go::resolveRequest` adds resolvers for `Bearer`, `Basic`, and `APIKey` (header or query placement) auth modes alongside the existing AWSv4 path.
- Tests updated: `interp_test.go`, `auth_test.go` now use canonical types and the Bruno loader.

### Step 7: Collection fuzzy picker overlay
**Status**: completed
**Timestamp**: 2026-05-08T15:10:00Z
- New `internal/tui/panes/picker.go`: `PickerModel` with `textinput.Model`, `matchScore` (name exact > prefix > contains; path basename > path-contains; `-1` = no match), `Update` returning `PickerContinue/Selected/Cancelled`, and `View` rendering a Catppuccin-styled box.
- `app.go`: `gc` chord opens picker; key dispatcher routes keys while `m.showPicker` is true. On select, expands the chosen collection in the tree, places the cursor on it, and runs `syncEnvPane`.
- `layout.go`: picker overlaid via `overlay()` helper at half-screen width.
- Test: `picker_test.go` exercises `matchScore` for empty/exact/prefix/contains/none cases.

### Step 8: jq response filter bar
**Status**: completed
**Timestamp**: 2026-05-08T15:20:00Z
- Added `github.com/itchyny/gojq` v0.12.19 to go.mod via `go get`.
- `internal/tui/panes/response.go`: new `filterMode`/`filterInput`/`filterQuery`/`filterBody`/`filterErr` state on `ResponseModel`. Pure helper `applyJQ(jsonBody, query) (string, error)` parses with `gojq.Parse`, runs the iterator, marshals the first result with indent, and recovers any panic into a returned error. `|` toggles filter mode; Enter applies and re-renders via the existing rebuild path (the filtered body is swapped into a cloned `*httpx.Response`); Esc clears. Bottom-hint area now shows the active filter, the error (when any), and the prompt.
- `Searching()` returns true while `filterMode` so the root key dispatch routes keys to the response pane.
- Tests: `response_test.go` covers `applyJQ` for empty/select/array-index/invalid-syntax/non-JSON cases.

### Step 9: Diagnostics overlay pane + status badge
**Status**: completed
**Timestamp**: 2026-05-08T15:25:00Z
- New `internal/tui/panes/diagnostics.go`: `DiagnosticsModel` with `Open/Close/Toggle/Up/Down/Selected`, `SeverityIcon`, and a Catppuccin-styled `View`. Empty state shows "No diagnostics".
- `app.go`: `m.diagnostics []canonical.Diagnostic` populated by `LoadCollections`. `gd` chord opens the modal. Modal absorbs `j/k`, `esc/q`, `enter` (Enter best-effort opens the file via `$EDITOR` reusing `openEnvInEditor`).
- `app.go::renderStatusBar` appends a `⚠ N` badge when `len(m.diagnostics) > 0`.
- `layout.go` overlays the modal when `m.showDiag`.
- Test: `diagnostics_test.go` validates rendered text + path:line + severity icon, plus the empty-state placeholder.

### Step 10: Docs + final polish
**Status**: completed
**Timestamp**: 2026-05-08T15:30:00Z
- `README.md`: Features section now mentions the pluggable loader interface, the jq filter, the diagnostics overlay; key bindings tables document `gc`, `gd`, and `|`. New "Configuration" section describes the `collections` key and notes `format = "..."` / `CollectionEntry` semantics.
- `make check` clean across the whole repo (fmt + vet + lint + tests).

### `make check` Status
After Steps 5–10: **all checks pass** (fmt, vet, lint, race-tested test suite).
