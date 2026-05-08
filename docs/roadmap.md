# brio Roadmap

brio is pre-1.0. Iterations are named by theme, not version number.

---

## Done: canonical model + plugin interface

**Goal**: decouple brio from Bruno-only `.bru` format; establish plugin seam for future loaders.

**Delivered**:
- `internal/canonical` — format-agnostic collection model (`Collection`, `Folder`, `Request`, `Scripts`, `AuthBlock`, `Diagnostic`)
- `CollectionLoader` plugin interface (`internal/plugins`)
- Bruno loader wrapping existing parser (no behavior change for Bruno users)
- Config `CollectionEntry{Path, Format}` (in-memory only at this stage)
- Fuzzy collection picker overlay (`gc`)
- jq response filter bar (`|`)
- Diagnostics overlay pane (`gd`) with status line badge
- `docs/adr/` — architectural decisions recorded

---

## Done: collection persistence + import modal

**Goal**: persist collections across sessions; let users add collections via a TUI import flow without editing config files manually.

**Delivered**:
- `[[collections]]` TOML table schema in `config.toml` (migrated from flat `[]string`; backward-compat read)
- `Config.AddCollection` / `Config.RemoveCollection` with abs-path dedupe
- `AutodetectLoader` opt-in interface — Bruno scans `preferences.json` + CWD
- Multi-step collection manager modal (`I` keybinding): plugin pick → path input / autodetect → multi-select candidates → confirm
- Empty-state startup: auto-opens import modal when no collections configured
- Bruno prefs fallback removed from CLI startup — autodetect is user-initiated only

---

## Current: Postman collection support

**Goal**: load Postman v2.1 collections via the plugin interface.

**Delivers**:
- `internal/plugins/postman` — JSON → canonical mapping
- goja JS runtime for Postman pre/post scripts
- `sahilm/fuzzy` (or equivalent) for picker if not already added
- Test fixtures and table-driven loader tests

---

## Next: Auth extensibility

**Goal**: make auth management first-class and format-agnostic, not AWS-only.

**Context**: brio currently hard-wires AWS SigV4 as the only auth mechanism in `internal/httpx`. Auth should be a pluggable, configurable layer so protected APIs using Bearer tokens, Basic auth, API keys, OAuth2, and custom credential-refresh hooks can all be configured without code changes.

**Planned work**:
- Define `Authenticator` interface in `internal/httpx` (or `internal/auth`)
- Built-in authenticators: Bearer, Basic, API key (header/query), AWS SigV4 (existing), no-op
- Hook-based authenticator: delegates to credential-refresh hooks (already exists in `internal/hooks`) — enables OAuth2, custom SSO, etc.
- Auth configured per-request → per-folder → per-collection (existing inheritance chain, extended to canonical)
- `AuthBlock` in canonical model extended with `APIKey` and hook-based auth fields
- Collection config can specify auth type + params without hard-coding credentials

**Open questions**:
- OAuth2 as a built-in, or hook-only? (hook-only is simpler and already works for AWS)
- Token caching: per-session only, or persist to keychain?

---

## Later: Theme system

**Goal**: make brio's visual style configurable.

**Delivers**:
- `Theme` interface in `internal/theme`
- Built-in themes: Catppuccin Macchiato (current default), Catppuccin Mocha, Tokyo Night
- User TOML theme override at `$XDG_CONFIG_HOME/brio/themes/*.toml`
- `theme = "<name>"` in config (placeholder field already present)

---

## Deferred / considering

- Insomnia / OpenAPI / HAR loaders
- Persisting jq filters per-request
- WebSocket / gRPC / GraphQL request types
- LSP-style live collection diagnostics (currently one-shot at load time)
