# Requirement: Persistent collection paths in config

## Problem

Running `brio` with no arguments currently auto-discovers collections from
Bruno's `preferences.json`. Users who don't have Bruno installed, or who want
a different set of collections than what Bruno has open, must pass paths on
every invocation.

There is no persistent way to declare "these are my collections" outside of
a shell alias.

## Desired behaviour

If `~/.config/brio/config.toml` contains a `collections` list, `brio` (with
no arguments) loads exactly those paths — Bruno preferences are not consulted.

```toml
collections = [
  "~/projects/api-gateway",
  "~/projects/payments",
]
```

Explicit CLI arguments always win over the config file.

## Resolution order (no-arg invocation)

1. `config.toml → collections` (if non-empty)
2. Bruno `preferences.json` (existing behaviour)
3. Error — no collections found

## Constraints

- `~` and `$ENV` vars must be expanded in each path
- Non-existent paths are skipped with a warning (not a fatal error)
- Settings modal (`gs`) should surface the list alongside hooks
