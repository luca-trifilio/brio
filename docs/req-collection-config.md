# Requirement: Persistent collection paths in config

## Problem

Running `brio` with no arguments currently auto-discovers collections from
Bruno's `preferences.json`. Users who don't have Bruno installed, or who want
a different set of collections than what Bruno has open, must pass paths on
every invocation.

There is no persistent way to declare "these are my collections" outside of
a shell alias.

## Desired behaviour

If `~/.config/brio/config.toml` contains a `[[collections]]` list, `brio` (with
no arguments) loads exactly those paths — Bruno preferences are not consulted.

```toml
[[collections]]
path = "~/projects/api-gateway"
format = "bruno"           # optional — auto-detected when omitted

[[collections]]
path = "~/projects/payments"
```

The legacy flat form `collections = ["..."]` continues to load. Saving (via
the import modal or `gs → c → s`) rewrites the file in the table form.

Explicit CLI arguments always win over the config file.

## Resolution order (no-arg invocation)

1. `config.toml → [[collections]]` (if non-empty)
2. Bruno `preferences.json` (existing behaviour)
3. Empty start — TUI opens with the import modal so the user can add a
   collection from inside brio

## Constraints

- `~` and `$ENV` vars must be expanded in each path
- Non-existent paths are skipped with a warning (not a fatal error)
- Settings modal (`gs`) shows the list alongside hooks
- Press `I` from any pane to open the collection-management modal directly
- Comments in `config.toml` are dropped on save (BurntSushi/toml limitation)
