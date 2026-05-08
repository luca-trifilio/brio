# PRD: brio v2 — Multi-format, Plugin-based API Collection TUI

## Problem Statement

brio is currently a capable terminal API client for Bruno collections, but it is tightly coupled to the Bruno file format and to a multi-collection-at-once mental model. Developers who use Postman or Insomnia collections have no equivalent keyboard-driven terminal tool. The internal model is Bruno-shaped from top to bottom, making it expensive to add new formats later. The community has no clear contribution surface for extending the tool with new integrations or themes.

## Solution

Introduce a format-agnostic canonical collection model that all format integrations (Bruno, Postman, Insomnia) translate into. Make each format integration a compiled-in Go plugin under `plugins/`, each satisfying a `CollectionLoader` interface. Add a collection registry with a fuzzy in-TUI switcher so the user always works with one collection at a time. Introduce inline `jq`-style response filtering, a diagnostics pane, and a themeing plugin system, all while keeping the single-binary, read-only, vim-style philosophy intact.

## User Stories

1. As a Postman user, I want brio to load my `collection.json` file, so that I can use a keyboard-driven terminal client without migrating to Bruno.
2. As an Insomnia user, I want brio to detect and load my Insomnia workspace, so that I can fire requests from the terminal without leaving my existing collection format.
3. As a developer, I want brio to auto-detect the collection format when I pass a path, so that I don't have to specify the format manually in the common case.
4. As a developer, I want to override the detected format via a `--format` flag, so that I can work with non-standard directory layouts.
5. As a developer, I want to configure the format for each registered collection in the TUI settings, so that I don't need to remember CLI flags.
6. As a developer, I want to register multiple collections in brio and switch between them instantly with a fuzzy picker, so that I can move between projects without restarting brio.
7. As a developer, I want the fuzzy collection picker to be accessible via a single keybinding, so that switching collections is as fast as switching buffers in vim.
8. As a developer, I want brio to open the fuzzy picker when launched with no arguments, so that I can quickly resume work on any registered collection.
9. As a developer, I want `brio /path` to still work as a one-shot launch, so that existing muscle memory and scripts are not broken.
10. As a developer, I want each collection to have its own history store, so that switching to a different project does not pollute my request history.
11. As a developer, I want brio to start with a clean slate when I switch collections, so that runtime variables from one project do not leak into another.
12. As a developer, I want all collection registry management (add, remove, rename) to live inside the TUI settings pane, so that I never need to manually edit config files.
13. As a developer, I want brio to execute Bearer token auth on Postman/Insomnia requests, so that I can fire authenticated requests without modifying the collection.
14. As a developer, I want brio to execute Basic auth on imported requests, so that username/password APIs work out of the box.
15. As a developer, I want brio to execute API Key auth on imported requests, so that key-based APIs work without manual header setup.
16. As a developer, I want AWS SigV4 to continue working exactly as today, so that my existing AWS collections are unaffected by the v2 migration.
17. As a developer, I want post-response scripts from Postman collections to set runtime variables, so that chained requests work the same way they do in Postman.
18. As a developer, I want pre-request scripts from Bruno collections to continue working as today, so that existing Bruno workflows are not broken.
19. As a developer, I want script execution to be powered by a shared JS runtime (goja) across all format plugins, so that script behavior is consistent regardless of the source format.
20. As a developer, I want to type a `jq` filter in the response pane and see the filtered result live, so that I can explore large JSON responses without leaving brio.
21. As a developer, I want syntax highlighting to continue working on filtered results, so that the response pane remains readable after applying a filter.
22. As a developer, I want a diagnostics pane accessible via `gd`, so that I can see all parse warnings, script errors, and hook failures in one place.
23. As a developer, I want parse failures for individual requests to appear in the diagnostics pane rather than aborting the collection load, so that one bad file does not block my work.
24. As a developer, I want hook errors to appear in the diagnostics pane, so that I can debug credential refresh failures without losing context.
25. As a community contributor, I want to write a new format integration as a Go package under `plugins/`, so that I can contribute Insomnia or OpenAPI support without touching core logic.
26. As a community contributor, I want a documented `CollectionLoader` interface, so that I know exactly what a format plugin must implement.
27. As a community contributor, I want to write a new theme as a Go package under `themes/`, so that I can contribute visual styles without touching core logic.
28. As a community contributor, I want my plugin or theme subdirectory listed in MAINTAINERS.md, so that I have clear ownership of my contribution area.
29. As a developer, I want the Catppuccin Macchiato theme to remain the default, so that the existing visual experience is unchanged for current users.
30. As a developer, I want variable interpolation (`{{var}}`) to work identically across all format integrations, so that environment variables resolve the same way regardless of the source format.

## Implementation Decisions

### Canonical Collection Model
- Introduce a new format-agnostic `canonical` package that defines `Collection`, `Folder`, `Request`, `Environment`, `AuthBlock`, `Body`, `Header`, `Param`, `Var`, and `Script` types with no Bruno-specific fields.
- All format plugins translate their native structures into the canonical model at load time.
- The Bruno plugin wraps the existing parser and `model` package, translating `BruDoc`-based types into canonical types.
- Bruno-specific fields (`BrunoConfig`, `CollectionDoc`, `BruDoc` on `Request`) are removed from the canonical model and remain internal to the Bruno plugin.

### Plugin Interface (`CollectionLoader`)
- Each format plugin implements a `CollectionLoader` interface with at minimum: `Name() string`, `Detect(path string) bool`, `Load(path string) (*canonical.Collection, error)`.
- Plugins register themselves at startup via an `init()`-style registry.
- All plugins live under `plugins/<format>/` in the monorepo.
- Format detection: brio tries each registered plugin's `Detect` in registration order; explicit `--format` flag or per-collection config overrides detection.

### Collection Registry & Switcher
- Config `collections` entries become structs with `path` and optional `format` fields (breaking config change from flat string list).
- The fuzzy collection picker is a new TUI overlay triggered by a keybinding (e.g. `gc`).
- `brio` (no args) opens the picker; `brio /path` bypasses the registry and loads directly.
- Switching collections resets all session state: active environment, runtime vars, response pane contents.
- History store is keyed by collection path; each collection gets its own history file.

### Authentication
- The canonical `AuthBlock` supports: `none`, `inherit`, `bearer`, `basic`, `apikey`, `awsv4`.
- The HTTP executor (`httpx`) is extended to apply `bearer`, `basic`, and `apikey` auth from the canonical block.
- OAuth2 is out of scope.

### Scripting
- Define a `ScriptContext` interface: inputs are the resolved request, the HTTP response, and the current var snapshot; output is a map of var mutations.
- Each format plugin provides a `ScriptRunner` that satisfies this interface.
- The shared JS runtime is `goja` (pure Go ES5/ES6). Both the Bruno plugin and the Postman plugin use goja internally.
- The Bruno plugin maps `bru.setVar` / `res.body` to canonical var mutations via goja.
- The Postman plugin maps `pm.environment.set` / `pm.response` to canonical var mutations via goja.

### Response Pane — jq Filtering
- Add a filter input bar at the bottom of the response pane (toggled by a keybinding, e.g. `|`).
- Use `gojq` (pure Go) to evaluate the filter against the raw response body.
- The filtered result replaces the displayed content; syntax highlighting is applied to the result.
- An empty or invalid filter falls back to the raw response; parse errors appear in the status bar.

### Diagnostics Pane
- A new toggleable pane accessible via `gd`.
- Collects: collection load warnings (skipped files, parse errors), script execution errors, hook errors, variable resolution warnings (unresolved `{{var}}`).
- Entries are timestamped and scoped to the active collection session.
- Cleared on collection switch.

### Theme Plugin Interface
- Define a `Theme` struct (palette of named colors) in a `themes/` package.
- Each theme is a Go package under `themes/<name>/` that exports a `Theme` value.
- The active theme is selected from config; falls back to Catppuccin Macchiato.

### Config Changes
- `collections` changes from `[]string` to `[]CollectionEntry{Path, Format string}`.
- Add `active_collection` string field (path of last active collection).
- Add `theme` string field.
- Migration: on first load, plain string entries in `collections` are silently upgraded to `CollectionEntry{Path: s}`.

## Testing Decisions

A good test verifies observable behavior through the module's public interface — it does not assert on internal state, private fields, or implementation details. Tests should remain valid through refactors that don't change behavior.

### Modules to test:
- **`canonical` model** — construction helpers, zero-value safety, auth inheritance resolution.
- **`plugins/bruno`** — round-trip: given a `.bru` fixture, assert the canonical `Request` fields. Existing corpus tests serve as prior art (`internal/parser/corpus_test.go`).
- **`plugins/postman`** — round-trip: given a `collection.json` fixture, assert canonical fields. Table-driven, one fixture per auth type and body type.
- **`plugins/insomnia`** — same pattern as Postman plugin.
- **`interp` (variable interpolation)** — existing tests are the prior art; extend for canonical `VarScope`.
- **`httpx` executor** — extend existing `executor_test.go` to cover bearer, basic, apikey auth injection.
- **`jq` filter layer** — given a JSON string and a filter expression, assert the output string. Test invalid filter fallback behavior.
- **Plugin registry** — assert that `Detect` picks the correct plugin for known directory layouts; assert explicit format override takes precedence.

### Out of scope for tests:
- TUI rendering (Bubble Tea model tests are integration-level; skip unless a specific bug warrants it).
- `goja` runtime internals — test only the canonical var mutations produced by script execution, not the JS runtime itself.

## Out of Scope

- Collection runner (executing all requests in a folder sequentially).
- OAuth2 (any flow).
- Writing to `.bru` or any other collection format files.
- External (non-Go) plugins.
- Full Postman `pm.*` API coverage — only the subset that maps to canonical var mutations.
- Full Insomnia feature parity beyond HTTP methods, headers, body, variables, and basic auth types.
- OpenAPI / Swagger import.
- Multi-collection side-by-side view.

## Further Notes

- The config schema change (`collections` from `[]string` to `[]CollectionEntry`) is a breaking change for existing `config.toml` files. A silent migration on first load is the least-friction path.
- `goja` adds a non-trivial binary size increase (~5MB). This is acceptable given the value it provides.
- The Bruno plugin should be the reference implementation for all other plugins — its tests, interface usage, and error handling patterns set the standard.
- The `CollectionLoader` interface should be considered stable once the first non-Bruno plugin ships; changes to it after that point require updating all plugins.
- Environment safety tiers (`●` safe · `▲` caution · `⚠` danger) and mutating-method blocking in production remain unchanged and are enforced at the executor level, not the plugin level.
