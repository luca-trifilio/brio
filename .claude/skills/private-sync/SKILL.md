---
name: private-sync
description: Use when syncing work across machines, pushing to the private remote, setting up brio on a new machine, managing what gets committed publicly vs privately, or understanding the dual-remote git setup.
---

# private-sync

Manage the dual-remote git setup that keeps personal workflow files private while publishing clean OSS commits.

## Remote Layout

| Remote | URL | Purpose |
|--------|-----|---------|
| `origin` | `git@github-personal:luca-trifilio/brio.git` | Public OSS repo — clean, no personal files |
| `private` | `git@github-personal:luca-trifilio/brio-private.git` | Private repo — everything, including `.claude/`, `docs/`, skills, tasks |

## Branch Strategy

- `main` on `origin` — public OSS branch; never contains `.claude/`, personal docs, or workflow files
- `sync` on `private` — tracks all personal content; has a relaxed `.gitignore` that includes `.claude/` and docs

The public `.gitignore` excludes `.claude/` and personal docs. The `sync` branch overrides this with a permissive `.gitignore` so those files are committed.

ADRs (`docs/adr/`) are the only docs committed to the public repo.

## Daily Workflow

Sync everything to the private remote frequently:

```sh
git push private
```

Publish to the public OSS remote only when a feature is ready:

```sh
git push origin
```

Never push `.claude/`, personal workflow docs, or skills to `origin`.

## New Machine Setup

Clone from the private remote to get all personal files:

```sh
git clone git@github-personal:luca-trifilio/brio-private.git
git checkout sync          # gets .claude/, docs/, skills, tasks
git remote add origin git@github-personal:luca-trifilio/brio.git
```

After setup both remotes are available: `git push private` for personal sync, `git push origin` for OSS publishing.

## What Lives Where

| Content | `origin` (public) | `private sync` |
|---------|:-----------------:|:--------------:|
| Source code, tests | yes | yes |
| `docs/adr/` | yes | yes |
| `.claude/` (skills, memory, hooks) | no | yes |
| `docs/` (non-ADR) | no | yes |
| Personal workflow files | no | yes |
| `.pi/` local-only dir | never | never (gitignored) |
